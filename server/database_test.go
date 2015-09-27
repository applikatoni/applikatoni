package main

import (
	"database/sql"
	"testing"
	"time"

	"bitbucket.org/liamstask/goose/lib/goose"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"
)

var dbConfigDirectory string = "./db"
var migrationsDirectory string = "./db/migrations"
var cleanStmts []string = []string{
	"DELETE FROM deployments;",
	"DELETE FROM log_entries;",
	"DELETE FROM users;",
}

func newTestDb(t *testing.T) *sql.DB {
	testConf, err := goose.NewDBConf(dbConfigDirectory, "test", "")
	checkErr(t, err)

	db, err := goose.OpenDBFromDBConf(testConf)
	checkErr(t, err)

	currentVersion, err := goose.EnsureDBVersion(testConf, db)
	checkErr(t, err)

	newestVersion, err := goose.GetMostRecentDBVersion(migrationsDirectory)
	checkErr(t, err)

	if currentVersion != newestVersion {
		t.Errorf("test DB not fully migrated. current version: %d, possible version: %d", currentVersion, newestVersion)
	}

	return db
}

func cleanCloseTestDb(db *sql.DB, t *testing.T) {
	for _, stmt := range cleanStmts {
		_, err := db.Exec(stmt)
		checkErr(t, err)
	}
	db.Close()
}

func buildUser(id int, name string) *models.User {
	return &models.User{
		Name:        name,
		Id:          id,
		AccessToken: "f00bardummytoken",
		AvatarUrl:   "http://www.github.com/avatars/avatar.png",
	}
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func buildDeployment(userId int) *models.Deployment {
	return &models.Deployment{
		UserId:          userId,
		CommitSha:       "f133742",
		Branch:          "master",
		Comment:         "Deploying a hotfix",
		ApplicationName: "flincOnRails",
		TargetName:      "production",
	}
}

func TestCreateDeployment(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	deployment := buildDeployment(9999)

	err := createDeployment(db, deployment)
	checkErr(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(1) FROM deployments").Scan(&count)
	checkErr(t, err)

	if count != 1 {
		t.Errorf("Wrong count of deployments in test DB")
	}

	if deployment.Id == 0 {
		t.Errorf("deployment has no ID after creation. got=%d", deployment.Id)
	}

	if deployment.State != models.DEPLOYMENT_NEW {
		t.Errorf("deployment has wrong State after creation. got=%s", deployment.State)
	}

	nullTime := time.Time{}
	if deployment.CreatedAt == nullTime {
		t.Errorf("deployment has wrong CreatedAt after creation. got=%s", deployment.CreatedAt)
	}
}

func TestUpdateDeploymentState(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	deployment := buildDeployment(9999)

	err := createDeployment(db, deployment)
	checkErr(t, err)

	err = updateDeploymentState(db, deployment, models.DEPLOYMENT_SUCCESSFUL)
	checkErr(t, err)

	var savedState string
	err = db.QueryRow("SELECT state FROM deployments WHERE id=?", deployment.Id).Scan(&savedState)
	checkErr(t, err)

	if savedState != string(models.DEPLOYMENT_SUCCESSFUL) {
		t.Errorf("deployment state not saved to db. got=%s", savedState)
	}

	if deployment.State != models.DEPLOYMENT_SUCCESSFUL {
		t.Errorf("deployment state not updated in struct. got=%s", deployment.State)
	}
}

func TestGetApplicationDeployments(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	firstDeployment := buildDeployment(9999)
	err := createDeployment(db, firstDeployment)
	checkErr(t, err)

	secondDeployment := buildDeployment(9999)
	err = createDeployment(db, secondDeployment)
	checkErr(t, err)

	application := &models.Application{Name: "flincOnRails"}

	deployments, err := getApplicationDeployments(db, application, 99)
	checkErr(t, err)

	if len(deployments) != 2 {
		t.Errorf("Wrong number of deployments returned. expected=%d, got=%d", 2, len(deployments))
	}

	if deployments[0].Id != secondDeployment.Id {
		t.Errorf("Deployments not in correct order. expected id=%d, got=%d", secondDeployment.Id, deployments[0].Id)
	}

	if deployments[1].Id != firstDeployment.Id {
		t.Errorf("Deployments not in correct order. expected id=%d, got=%d", firstDeployment.Id, deployments[1].Id)
	}

	deployments, err = getApplicationDeployments(db, application, 1)
	checkErr(t, err)

	if len(deployments) != 1 {
		t.Errorf("Wrong number of deployments returned. expected=%d, got=%d", 1, len(deployments))
	}
}

func TestGetDeployment(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	deployment := buildDeployment(9999)
	err := createDeployment(db, deployment)
	checkErr(t, err)

	savedDeployment, err := getDeployment(db, deployment.Id)
	checkErr(t, err)

	if savedDeployment.Id != deployment.Id {
		t.Errorf("wrong id. got=%d, want=%d", savedDeployment.Id, deployment.Id)
	}
	if savedDeployment.ApplicationName != deployment.ApplicationName {
		t.Errorf("wrong ApplicationName. got=%s want=%s", savedDeployment.ApplicationName, deployment.ApplicationName)
	}
	if savedDeployment.CommitSha != deployment.CommitSha {
		t.Errorf("wrong sha. got=%s want=%s", savedDeployment.CommitSha, deployment.CommitSha)
	}
	if savedDeployment.Branch != deployment.Branch {
		t.Errorf("wrong sha. got=%s want=%s", savedDeployment.CommitSha, deployment.CommitSha)
	}
	if savedDeployment.CreatedAt.UTC() != deployment.CreatedAt.UTC() {
		t.Errorf("wrong timestamp. got=%s want=%s", savedDeployment.CreatedAt.UTC(), deployment.CreatedAt.UTC())
	}
}

func TestGetLastTargetDeployment(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	deployments := []*models.Deployment{
		{
			UserId:          1,
			CommitSha:       "f133742",
			Branch:          "master",
			Comment:         "one",
			ApplicationName: "app",
			TargetName:      "prod",
		},
		{
			UserId:          1,
			CommitSha:       "f133743",
			Branch:          "master",
			Comment:         "one-staging",
			ApplicationName: "app",
			TargetName:      "staging",
		},
		{
			UserId:          1,
			CommitSha:       "f133742",
			Branch:          "master",
			Comment:         "two",
			ApplicationName: "app",
			TargetName:      "prod",
		},
	}

	for _, d := range deployments {
		err := createDeployment(db, d)
		checkErr(t, err)
	}

	last, err := getLastTargetDeployment(db, "prod")
	checkErr(t, err)

	if last == nil {
		t.Errorf("returned deployment is nil")
	}
	if last.Id != deployments[0].Id {
		t.Errorf("wrong last deployment id. expected=%d, got=%d",
			deployments[0].Id, last.Id)
	}

	last, err = getLastTargetDeployment(db, "staging")
	checkErr(t, err)

	if last == nil {
		t.Errorf("returned deployment is nil")
	}
	if last.Id != deployments[1].Id {
		t.Errorf("wrong last deployment id. expected=%d, got=%d",
			deployments[1].Id, last.Id)
	}
}

func TestCreateLogEntry(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	entry := deploy.LogEntry{
		DeploymentId: 99,
		Origin:       "production.server.com",
		EntryType:    deploy.COMMAND_START,
		Message:      "bundle exec rake db:migrate",
		Timestamp:    time.Now(),
	}

	err := createLogEntry(db, &entry)
	checkErr(t, err)

	if entry.Id == 0 {
		t.Errorf("entry has no ID after creation")
	}

	var count int
	err = db.QueryRow("SELECT COUNT(1) FROM log_entries").Scan(&count)
	checkErr(t, err)

	if count != 1 {
		t.Errorf("wrong count of log_entries. want=%d, got=%d", 1, count)
	}
}

func TestGetDeploymentLogEntries(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	deployment := &models.Deployment{Id: 99}
	firstEntry := deploy.LogEntry{
		DeploymentId: deployment.Id,
		Origin:       "production.server.com",
		EntryType:    deploy.COMMAND_START,
		Message:      "bundle exec rake db:migrate",
		Timestamp:    time.Now(),
	}
	err := createLogEntry(db, &firstEntry)
	checkErr(t, err)

	secondEntry := deploy.LogEntry{
		DeploymentId: deployment.Id,
		Origin:       "production.server.com",
		EntryType:    deploy.COMMAND_SUCCESS,
		Message:      "bundle exec rake db:migrate",
		Timestamp:    time.Now(),
	}
	err = createLogEntry(db, &secondEntry)
	checkErr(t, err)

	entries, err := getDeploymentLogEntries(db, deployment)
	checkErr(t, err)

	if len(entries) != 2 {
		t.Errorf("wrong length of logentries. want=%d, got=%d", 2, len(entries))
	}

	for _, e := range entries {
		if e.Origin != firstEntry.Origin {
			t.Errorf("wrong Origin saved. want=%s, got=%s", firstEntry.Origin, e.Origin)
		}
		if e.Message != firstEntry.Message {
			t.Errorf("wrong message saved. want=%s, got=%s", firstEntry.Message, e.Message)
		}
	}

	if entries[0].Id != firstEntry.Id {
		t.Error("wrong order of entries.")
	}

	if entries[1].Id != secondEntry.Id {
		t.Error("wrong order of entries.")
	}
}

func TestNewLogEntrySaver(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	deployment := &models.Deployment{Id: 99}
	ch := make(chan deploy.LogEntry)

	go func() {
		ch <- deploy.LogEntry{
			DeploymentId: deployment.Id,
			Origin:       "production.server.com",
			EntryType:    deploy.COMMAND_START,
			Message:      "bundle exec rake db:migrate",
			Timestamp:    time.Now(),
		}
		ch <- deploy.LogEntry{
			DeploymentId: deployment.Id,
			Origin:       "production.server.com",
			EntryType:    deploy.COMMAND_SUCCESS,
			Message:      "bundle exec rake db:migrate",
			Timestamp:    time.Now(),
		}

		close(ch)
	}()

	fn := newLogEntrySaver(db)
	fn(ch)

	entries, err := getDeploymentLogEntries(db, deployment)
	checkErr(t, err)

	if len(entries) != 2 {
		t.Errorf("not enough log entries saved. want=%d, got=%d", 2, len(entries))
	}
}

func TestCreateUser(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	user := buildUser(12345, "mrnugget")

	err := createUser(db, user)
	checkErr(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(1) FROM users WHERE id = ?", user.Id).Scan(&count)
	checkErr(t, err)

	if count != 1 {
		t.Errorf("wrong count of users. want=%d, got=%d", 1, count)
	}
}

func TestCreateUserApiToken(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	user := buildUser(12345, "mrnugget")

	err := createUser(db, user)
	checkErr(t, err)

	if user.ApiToken == "" {
		t.Errorf("Expected user to have an ApiToken after creation")
	}
}

func TestGetUser(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	user := buildUser(12345, "mrnugget")

	err := createUser(db, user)
	checkErr(t, err)

	newUser, err := getUser(db, user.Id)
	checkErr(t, err)

	if newUser.Id != user.Id {
		t.Errorf("wrong id. want=%d, got=%d", user.Id, newUser.Id)
	}

	if newUser.Name != user.Name {
		t.Errorf("wrong Name. want=%d, got=%d", user.Name, newUser.Name)
	}

	if newUser.AccessToken != user.AccessToken {
		t.Errorf("wrong AccessToken. want=%d, got=%d", user.AccessToken, newUser.AccessToken)
	}

	if newUser.AvatarUrl != user.AvatarUrl {
		t.Errorf("wrong AvatarUrl. want=%d, got=%d", user.AvatarUrl, newUser.AvatarUrl)
	}

	if newUser.ApiToken != user.ApiToken {
		t.Errorf("wrong ApiToken. want=%d, got=%d", user.ApiToken, newUser.ApiToken)
	}
}

func TestGetOrCreateUser(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	user := buildUser(12345, "mrnugget")

	err := getOrCreateUser(db, user)
	checkErr(t, err)

	err = getOrCreateUser(db, user)
	checkErr(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(1) FROM users WHERE id = ?", user.Id).Scan(&count)
	checkErr(t, err)
	if count != 1 {
		t.Errorf("wrong count of users. want=%d, got=%d", 1, count)
	}
}

func TestLoadDeploymentsUsers(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	userOne := buildUser(12345, "mrnugget")
	err := getOrCreateUser(db, userOne)
	checkErr(t, err)

	userTwo := buildUser(56789, "fabrik42")
	err = getOrCreateUser(db, userTwo)
	checkErr(t, err)

	deploymentOne := buildDeployment(userOne.Id)
	err = createDeployment(db, deploymentOne)
	checkErr(t, err)

	deploymentTwo := buildDeployment(userTwo.Id)
	err = createDeployment(db, deploymentTwo)
	checkErr(t, err)

	s := []*models.Deployment{deploymentOne, deploymentTwo}
	err = loadDeploymentsUsers(db, s)
	checkErr(t, err)

	if deploymentOne.User == nil {
		t.Errorf("pointer to user is still nil")
	}
	if deploymentOne.User.Id != userOne.Id {
		t.Errorf("id of loaded user is wrong. want=%d, got=%d", userOne.Id, deploymentOne.User.Id)
	}
	if deploymentTwo.User == nil {
		t.Errorf("pointer to user is still nil")
	}
	if deploymentTwo.User.Id != userTwo.Id {
		t.Errorf("id of loaded user is wrong. want=%d, got=%d", userTwo.Id, deploymentTwo.User.Id)
	}
}

func TestCreateDeploymentWithActiveDeployments(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	deployment := buildDeployment(9999)

	err := createDeployment(db, deployment)
	checkErr(t, err)

	err = updateDeploymentState(db, deployment, models.DEPLOYMENT_ACTIVE)
	checkErr(t, err)

	newDeployment := buildDeployment(9999)

	err = createDeployment(db, newDeployment)
	if err != ErrDeployInProgress {
		t.Errorf("createDeployment didnt fail with correct error: %s", err)
	}
}

func TestGetDailyDigestDeployments(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	a := &models.Application{Name: "awesomeDB"}
	targetName := "production"
	since := time.Now().Add(-24 * time.Hour)

	stmt := `INSERT INTO
	deployments
	(user_id, application_name, target_name, commit_sha, branch, comment, state, created_at)
	VALUES
	(?, ?, ?, ?, ?, ?, ?, ?);`

	existingDeployments := []struct {
		applicationName string
		targetName      string
		createdAt       time.Time
		state           models.DeploymentState
	}{
		// should be included
		{"awesomeDB", "production", time.Now().Add(-12 * time.Hour), models.DEPLOYMENT_SUCCESSFUL},
		// these should not be included
		{"awesomeDB", "production", time.Now().Add(-45 * time.Hour), models.DEPLOYMENT_SUCCESSFUL},
		{"awesomeDB", "staging", time.Now().Add(-10 * time.Hour), models.DEPLOYMENT_SUCCESSFUL},
		{"awesomeDB", "production", time.Now().Add(-12 * time.Hour), models.DEPLOYMENT_FAILED},
	}

	for _, ed := range existingDeployments {
		_, err := db.Exec(stmt, 9999, ed.applicationName, ed.targetName,
			"f00b4r", "master", "foo", string(ed.state), ed.createdAt)
		checkErr(t, err)
	}

	deployments, err := getDailyDigestDeployments(db, a, targetName, since)
	checkErr(t, err)

	if len(deployments) != 1 {
		t.Errorf("getDailyDigestDeployments wrong number of deployments: %d", len(deployments))
	}
}

func TestFailUnfinishedDeployments(t *testing.T) {
	db := newTestDb(t)
	defer cleanCloseTestDb(db, t)

	stmt := `INSERT INTO
	deployments
	(user_id, application_name, target_name, commit_sha, branch, comment, state, created_at)
	VALUES
	(?, ?, ?, ?, ?, ?, ?, ?);`

	states := []models.DeploymentState{
		models.DEPLOYMENT_NEW,
		models.DEPLOYMENT_ACTIVE,
		models.DEPLOYMENT_FAILED,
		models.DEPLOYMENT_SUCCESSFUL,
	}

	for _, s := range states {
		_, err := db.Exec(stmt, 9999, "awesomeDB", "production",
			"f00b4r", "master", "foo", string(s), time.Now())
		checkErr(t, err)
	}

	err := failUnfinishedDeployments(db)
	checkErr(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(1) FROM deployments WHERE state = 'failed'").Scan(&count)
	checkErr(t, err)
	if count != 3 {
		t.Errorf("wrong count of failed deployments. want=%d, got=%d", 3, count)
	}

	err = db.QueryRow("SELECT COUNT(1) FROM deployments WHERE state = 'successful'").Scan(&count)
	checkErr(t, err)
	if count != 1 {
		t.Errorf("wrong count of successful deployments. want=%d, got=%d", 1, count)
	}
}
