package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"bitbucket.org/liamstask/goose/lib/goose"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/applikatoni/applikatoni/deploy"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

const VERSION = "1.1.0"

var (
	outputVersion         = flag.Bool("v", false, "output the version of Applikatoni")
	configurationFilePath = flag.String("conf", "configuration.json", "path to configuration file")
	port                  = flag.String("port", ":8080", "port to listen on")
	databasePath          = flag.String("db", "./db/development.db", "path to sqlite3 database file")
	templatesPath         = flag.String("templates", "./assets/templates", "path to template files")
	env                   = flag.String("env", "development", "environment applikatoni is used in")
	dbConfDir             = flag.String("dbconfdir", "./db", "path to directory of dbconf.yml")
	migrationDir          = flag.String("migrationdir", "./db/migrations", "path to migrations files")
)

var (
	logRouter    *deploy.LogRouter
	config       *Configuration
	db           *sql.DB
	sessionStore *sessions.CookieStore
	templates    map[string]*template.Template
	oauthCfg     *oauth2.Config
	killRegistry *KillRegistry
)

var (
	sessionName    = "applikatonisession"
	templatesFiles = [][]string{
		{"layout.tmpl", "hogan_templates.tmpl", "partials.tmpl", "home.tmpl"},
		{"layout.tmpl", "hogan_templates.tmpl", "partials.tmpl", "toni_configuration.tmpl"},
		{"layout.tmpl", "hogan_templates.tmpl", "partials.tmpl", "application.tmpl"},
		{"layout.tmpl", "hogan_templates.tmpl", "partials.tmpl", "deployments.tmpl"},
		{"layout.tmpl", "hogan_templates.tmpl", "partials.tmpl", "deployment.tmpl"},
	}
)

func main() {
	flag.Parse()

	if *outputVersion {
		fmt.Println(VERSION)
		return
	}

	var err error
	config, err = readConfiguration(*configurationFilePath)
	if err != nil {
		log.Fatal("could not read configuration", err)
	}

	templates, err = parseTemplates(*templatesPath, templatesFiles)
	if err != nil {
		log.Fatal("Parsing templates failed", err)
	}

	db, err = sql.Open("sqlite3", *databasePath)
	if err != nil {
		log.Fatal("could not open sqlite3 database file", err)
	}
	defer db.Close()

	migrated, err := isDBMigrated(db)
	if err != nil {
		log.Fatal("could not check if database is migrated. Error: ", err)
	}
	if !migrated {
		log.Fatal("please migrated to newest database version")
	}

	// If there are deployments in state 'new'/'active' when booting up
	// Applikatoni probably crashed with a deployment running. Set these to
	// 'failed' so we can start other deployments.
	err = failUnfinishedDeployments(db)
	if err != nil {
		log.Fatal("setting unfinished deployments to 'failed' failed", err)
	}

	oauthCfg = &oauth2.Config{
		ClientID:     config.GitHubClientId,
		ClientSecret: config.GitHubClientSecret,
		Scopes:       []string{"user", "repo"},
		Endpoint:     github.Endpoint,
	}

	// Setup the killRegistry to connect deployment managers to the kill button
	killRegistry = NewKillRegistry()

	// Run the daily digest sending in the background
	go SendDailyDigests(db)

	// Setup session store
	sessionStore = sessions.NewCookieStore([]byte(config.SessionSecret))

	// Initialize global LogRouter
	logRouter = deploy.NewLogRouter()
	defer logRouter.Stop()
	logRouter.Start()

	// Setup a basic listener that prints the logs of all deployments
	logRouter.SubscribeAll(deploy.ConsoleLogger)
	// Setup the listener that persists all log entries
	logRouter.SubscribeAll(newLogEntrySaver(db))
	// Setup the bugsnag deployment tracking
	logRouter.SubscribeAll(newBugsnagNotifier(db))
	// Setup the flowdock deployment notifcation
	logRouter.SubscribeAll(newFlowdockNotifier(db))
	// Setup the new relic deployment notifcation
	logRouter.SubscribeAll(newNewRelicNotifier(db))
	// Setup the Slack deployment notifcation
	logRouter.SubscribeAll(newSlackNotifier(db))

	// Setup the router and the routes
	r := mux.NewRouter()

	// Assets
	fsServer := http.FileServer(http.Dir("assets/"))
	assetsServer := http.StripPrefix("/assets/", fsServer)

	r.PathPrefix("/assets/").Handler(assetsServer)
	r.Handle("/favicon.ico", fsServer)

	// OAuth & Login
	r.HandleFunc("/oauth2/authorize", oauth2authorizeHandler)
	r.HandleFunc("/oauth2/callback", oauth2callbackHandler)
	r.HandleFunc("/oauth2/logout", oauth2logoutHandler)

	// Application
	r.HandleFunc("/{application}/deployments", requireAuthorizedUser(createDeploymentHandler)).Methods("POST")
	r.HandleFunc("/{application}/deployments", requireAuthorizedUser(listDeploymentsHandler)).Methods("GET")
	r.HandleFunc("/{application}/deployments/{deploymentId}", requireAuthorizedUser(deploymentHandler)).Methods("GET")
	r.HandleFunc("/{application}/deployments/{deploymentId}/log", requireAuthorizedUser(deploymentWsHandler)).Methods("GET")
	r.HandleFunc("/{application}/deployments/{deploymentId}/kill", requireAuthorizedUser(killDeploymentHandler)).Methods("POST")
	r.HandleFunc("/{application}/pulls", requireAuthorizedUser(pullRequestsHandler)).Methods("GET")
	r.HandleFunc("/{application}/branches", requireAuthorizedUser(branchesHandler)).Methods("GET")
	r.HandleFunc("/{application}/diff", requireAuthorizedUser(diffHandler)).Methods("GET")
	r.HandleFunc("/{application}/toni", requireAuthorizedUser(toniConfigurationHandler))
	r.HandleFunc("/{application}", requireAuthorizedUser(applicationHandler))

	// GET /
	r.HandleFunc("/", authenticate(homeHandler))

	log.Printf("Listening on localhost%s\n", *port)
	err = http.ListenAndServe(*port, handlers.LoggingHandler(os.Stdout, r))
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func isDBMigrated(db *sql.DB) (bool, error) {
	dbconf, err := goose.NewDBConf(*dbConfDir, *env, "")
	if err != nil {
		return false, err
	}

	currentVersion, err := goose.EnsureDBVersion(dbconf, db)
	if err != nil {
		return false, err
	}

	newestVersion, err := goose.GetMostRecentDBVersion(*migrationDir)
	if err != nil {
		return false, err
	}

	if currentVersion != newestVersion {
		return false, nil
	}

	return true, nil
}
