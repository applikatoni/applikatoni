package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v2"

	"golang.org/x/oauth2"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type contextKey int

const (
	CurrentUser contextKey = iota + 1
	CurrentApplication
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)

	renderTemplate(w, "home.tmpl", map[string]interface{}{
		"Applications": config.Applications,
		"currentUser":  currentUser,
	})
}

func applicationHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	application := getCurrentApplication(r)

	deployments, err := getRecentApplicationDeployments(db, application)
	if err != nil {
		log.Println("error loading deployments", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = loadDeploymentsUsers(db, deployments)
	if err != nil {
		log.Println("error loading the users of the deployments", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "application.tmpl", map[string]interface{}{
		"Applications": config.Applications,
		"Application":  application,
		"Deployments":  deployments,
		"currentUser":  currentUser,
	})
}

func toniConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	application := getCurrentApplication(r)

	var host string
	if r.TLS != nil || config.SSLEnabled {
		host = "https://" + r.Host
	} else {
		host = "http://" + r.Host
	}

	stages := make(map[string][]models.DeploymentStage)
	for _, t := range application.Targets {
		stages[t.Name] = t.DefaultStages
	}

	toniConfig := struct {
		Host        string                              `yaml:"host"`
		Application string                              `yaml:"application"`
		ApiToken    string                              `yaml:"api_token"`
		Stages      map[string][]models.DeploymentStage `yaml:"stages"`
	}{
		Host:        host,
		Application: application.Name,
		ApiToken:    currentUser.ApiToken,
		Stages:      stages,
	}

	configContent, err := yaml.Marshal(&toniConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	renderTemplate(w, "toni_configuration.tmpl", map[string]interface{}{
		"Applications":  config.Applications,
		"Application":   application,
		"currentUser":   currentUser,
		"configContent": string(configContent),
	})
}

func pullRequestsHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	application := getCurrentApplication(r)

	ghClient := NewGitHubClient(currentUser)
	pulls, err := ghClient.GetPullRequests(application)
	if err != nil {
		log.Println("error loading pull requests", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js, err := json.Marshal(pulls)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func branchesHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	application := getCurrentApplication(r)

	ghClient := NewGitHubClient(currentUser)
	branches, err := ghClient.GetBranches(application)
	if err != nil {
		log.Println("error loading branches", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(branches)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func diffHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	application := getCurrentApplication(r)

	targetName := r.URL.Query().Get("target")
	sha := r.URL.Query().Get("sha")

	if targetName == "" || sha == "" {
		http.Error(w, "target or sha missing", 422)
		return
	}

	d, err := getLastTargetDeployment(db, application, targetName)
	if err != nil {
		log.Println("getLastTargetDeployment failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if d == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(204)
		return
	}

	ghClient := NewGitHubClient(currentUser)
	diff, err := ghClient.Compare(application, d.CommitSha, sha)
	if err != nil {
		log.Println("error loading diff from github", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(diff)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
	return
}

func createDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	application := getCurrentApplication(r)

	target, err := findTarget(application, r.FormValue("target"))
	if err != nil {
		log.Printf("error: %s\n", err)
		http.NotFound(w, r)
		return
	}

	if !target.IsDeployer(currentUser.Name) {
		http.Error(w, "not authorized to deploy to this target", 403)
		return
	}

	comment := r.FormValue("comment")
	if comment == "" {
		http.Error(w, "comment is empty", 422)
		return
	}

	commitSha := r.FormValue("commitsha")
	if !isValidCommitSha(commitSha) {
		http.Error(w, "invalid commit sha", 422)
		return
	}

	formStages := r.Form["stages[]"]
	if len(formStages) == 0 {
		http.Error(w, "no stages selected", 422)
		return
	}

	stages := []models.DeploymentStage{}
	for _, fs := range formStages {
		stages = append(stages, models.DeploymentStage(fs))
	}

	if !target.AreValidStages(stages) {
		msg := "stages have wrong order or contain invalid stages. Available stages: %v"
		http.Error(w, fmt.Sprintf(msg, target.AvailableStages), 422)
		return
	}

	deployment := &models.Deployment{
		UserId:          currentUser.Id,
		CommitSha:       commitSha,
		Branch:          r.FormValue("branch"),
		Comment:         r.FormValue("comment"),
		ApplicationName: application.Name,
		TargetName:      target.Name,
	}

	err = createDeployment(db, deployment)
	if err != nil {
		log.Println("Could not save to database", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	eventHub.Publish(deployment.State, deployment)
	killChan := killRegistry.Add(deployment.Id)

	deploymentConfig := models.NewDeploymentConfig(deployment, target, stages)
	manager, err := deploy.NewManager(deploymentConfig, logRouter, killChan)
	if err != nil {
		log.Println("Could not build Manager", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	manager.AnnounceStart()

	err = updateDeploymentState(db, deployment, models.DEPLOYMENT_ACTIVE)
	if err != nil {
		log.Println("Could not update deployment state")
		killRegistry.Remove(deployment.Id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	eventHub.Publish(models.DEPLOYMENT_ACTIVE, deployment)

	go func() {
		newState := models.DEPLOYMENT_SUCCESSFUL
		err = manager.Start()
		if err != nil {
			newState = models.DEPLOYMENT_FAILED
		}

		err = updateDeploymentState(db, deployment, newState)
		if err != nil {
			log.Println("Could not update deployment state")
		} else {
			eventHub.Publish(newState, deployment)
		}

		killRegistry.Remove(deployment.Id)
	}()

	http.Redirect(w, r, deploymentUrl(application, deployment), http.StatusSeeOther)
}

func killDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["deploymentId"])
	if err != nil {
		log.Println("error converting ID passed to server", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	killChan, err := killRegistry.Get(id)
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}

	killChan <- struct{}{}
}

func listDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	application := getCurrentApplication(r)
	target, err := getTarget(application, r.URL.Query().Get("target"))

	var deployments []*models.Deployment

	if err != nil {
		deployments, err = getAllApplicationDeployments(db, application)
	} else {
		deployments, err = getApplicationDeploymentsByTarget(db, application, target)
	}

	if err != nil {
		log.Println("error loading deployments", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = loadDeploymentsUsers(db, deployments)
	if err != nil {
		log.Println("error loading the users of the deployments", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "deployments.tmpl", map[string]interface{}{
		"Applications":   config.Applications,
		"Application":    application,
		"Deployments":    deployments,
		"currentUser":    currentUser,
		"selectedTarget": target,
	})
}

func deploymentHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := getCurrentUser(r)
	application := getCurrentApplication(r)

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["deploymentId"])
	if err != nil {
		log.Println("error converting ID passed to server", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deployment, err := getDeployment(db, id)
	if err != nil {
		log.Println("error loading deployment", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	deploymentUser, err := getUser(db, deployment.UserId)
	if err != nil {
		log.Println("error loading deployment user", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	deployment.User = deploymentUser

	logEntries, err := getDeploymentLogEntries(db, deployment)
	if err != nil {
		log.Println("error loading logentries", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "deployment.tmpl", map[string]interface{}{
		"Applications": config.Applications,
		"Application":  application,
		"Deployment":   deployment,
		"LogEntries":   logEntries,
		"currentUser":  currentUser,
		"Host":         r.Host,
	})
}

func deploymentWsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["deploymentId"])
	if err != nil {
		log.Println("error converting ID passed to server", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deployment, err := getDeployment(db, id)
	if err != nil {
		log.Println("error loading deployment", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	upgrader := &websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("error upgrading the connection to websocket", err)
		return
	}

	go keepWsAlive(ws)

	doneStreaming := make(chan struct{})

	err = logRouter.Subscribe(id, makeWebsocketListener(ws, doneStreaming))
	if err == deploy.ErrNoDeployment {
		logEntries, err := getDeploymentLogEntries(db, deployment)
		if err != nil {
			log.Println("error loading logentries", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		go streamLogEntries(ws, doneStreaming, logEntries)
	}

	<-doneStreaming
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	ws.WriteMessage(websocket.CloseMessage, closeMsg)
	ws.Close()
}

func oauth2authorizeHandler(w http.ResponseWriter, r *http.Request) {
	url := oauthCfg.AuthCodeURL(config.Oauth2StateString)
	http.Redirect(w, r, url, http.StatusFound)
}

func oauth2callbackHandler(w http.ResponseWriter, r *http.Request) {
	// Check if state is the same as our saved state string
	state := r.FormValue("state")
	if state != config.Oauth2StateString {
		log.Println("oauth2 state string does not match")
		http.Error(w, "oauth2 state string does not match", http.StatusInternalServerError)
		return
	}

	//Get the code from the response
	code := r.FormValue("code")

	// Exchange the received code for a token
	token, err := oauthCfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Println("could not exchange code for access token", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := &models.User{AccessToken: token.AccessToken}
	ghClient := NewGitHubClient(user)
	err = ghClient.UpdateUser(user)
	if err != nil {
		log.Println("could not fetch user information from github: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = getOrCreateUser(db, user)
	if err != nil {
		log.Println("insertUser failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := sessionStore.Get(r, sessionName)
	session.Values["user_id"] = user.Id
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func oauth2logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, sessionName)
	delete(session.Values, "user_id")
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func loadUserFromSession(r *http.Request) (*models.User, error) {
	session, _ := sessionStore.Get(r, sessionName)

	if id, ok := session.Values["user_id"].(int); ok {
		user, err := getUser(db, id)
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	return nil, nil
}

func loadUserWithApiToken(r *http.Request) (*models.User, error) {
	token := r.Header.Get("X-Api-Token")
	if token == "" {
		return nil, nil
	}

	user, err := getUserByApiToken(db, token)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func getCurrentUser(r *http.Request) *models.User {
	u := context.Get(r, CurrentUser)
	if u != nil {
		return u.(*models.User)
	}
	return nil
}

func getCurrentApplication(r *http.Request) *models.Application {
	a := context.Get(r, CurrentApplication)
	if a != nil {
		return a.(*models.Application)
	}
	return nil
}

func getTarget(a *models.Application, t string) (*models.Target, error) {
	for _, i := range a.Targets {
		if i.Name == t {
			return i, nil
		}
	}
	return nil, errors.New("target not found")
}

func findApplication(name string) (*models.Application, error) {
	for _, a := range config.Applications {
		if a.Name == name {
			return a, nil
		}
	}
	return nil, errors.New("application not found")
}

func findTarget(a *models.Application, name string) (*models.Target, error) {
	for _, t := range a.Targets {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, errors.New("target not found")
}

func deploymentUrl(a *models.Application, d *models.Deployment) string {
	return fmt.Sprintf("/%s/deployments/%d", a.Name, d.Id)
}

func makeWebsocketListener(ws *websocket.Conn, done chan struct{}) deploy.Listener {
	return func(logs <-chan deploy.LogEntry) {
		defer func() {
			done <- struct{}{}
		}()
		for entry := range logs {
			err := ws.WriteJSON(entry)
			if err != nil {
				log.Printf("error writing to websocket: %s. (remote address=%s)\n", err, ws.RemoteAddr())
				return
			}
		}
	}
}

func streamLogEntries(ws *websocket.Conn, done chan struct{}, logs []*deploy.LogEntry) {
	defer func() {
		done <- struct{}{}
	}()
	for _, entry := range logs {
		err := ws.WriteJSON(entry)
		if err != nil {
			return
		}
	}
}

func isValidCommitSha(sha string) bool {
	validSha := regexp.MustCompile(`^[0-9a-f]{40}$`)

	return validSha.MatchString(sha)
}

func keepWsAlive(ws *websocket.Conn) {
	// We repeatedly read from the websocket connections and discard
	// the reader in order to process the underlying ping/pong messages
	// of the websocket connection
	for {
		_, _, err := ws.NextReader()
		if err != nil {
			ws.Close()
			break
		}
	}
}
