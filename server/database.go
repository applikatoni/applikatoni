package main

import (
	"errors"
	"log"
	"strings"
	"time"

	"bitbucket.org/liamstask/goose/lib/goose"

	"github.com/pborman/uuid"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"

	"database/sql"
)

const (
	deploymentStmt                     = `SELECT id, user_id, application_name, target_name, commit_sha, branch, comment, state, created_at FROM deployments WHERE deployments.id = ?`
	deploymentInsertStmt               = `INSERT INTO deployments (user_id, application_name, target_name, commit_sha, branch, comment, state, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`
	deploymentUpdateStateStmt          = `UPDATE deployments SET state = ? WHERE deployments.id = ?`
	deploymentFailUnfinishedStmt       = `UPDATE deployments SET state = ? WHERE deployments.state = ? OR deployments.state = ?`
	lastTargetDeploymentStmt           = `SELECT id, user_id, application_name, target_name, commit_sha, branch, comment, state, created_at FROM deployments WHERE deployments.state = ? AND deployments.application_name = ? AND deployments.target_name = ? ORDER BY created_at DESC LIMIT 1`
	applicationDeploymentsStmt         = `SELECT id, user_id, target_name, commit_sha, branch, comment, state, created_at FROM deployments WHERE deployments.application_name = ? ORDER BY created_at DESC LIMIT ?`
	applicationDeploymentsByTargetStmt = `SELECT id, user_id, target_name, commit_sha, branch, comment, state, created_at FROM deployments WHERE deployments.application_name = ? AND deployments.target_name = ? ORDER BY created_at DESC`
	logEntryInsertStmt                 = `INSERT INTO log_entries (deployment_id, entry_type, origin, message, timestamp, created_at) VALUES (?, ?, ?, ?, ?, ?);`
	deploymentLogEntriesStmt           = `SELECT id, deployment_id, entry_type, origin, message, timestamp FROM log_entries WHERE log_entries.deployment_id = ? ORDER BY timestamp ASC`
	userInsertStmt                     = `INSERT INTO users(id, name, access_token, avatar_url, api_token) VALUES(?, ?, ?, ?, ?);`
	userUpdateStmt                     = `UPDATE users SET access_token = ?, avatar_url = ? WHERE id = ?;`
	userStmt                           = `SELECT id, name, access_token, avatar_url, api_token FROM users WHERE id = ?;`
	userApiTokenStmt                   = `SELECT id, name, access_token, avatar_url, api_token FROM users WHERE api_token = ?;`
	activeDeploymentsStmt              = `SELECT state FROM deployments WHERE application_name = ? AND target_name = ? AND state = 'active' LIMIT 1;`
	dailyDigestDeploymentsStmt         = `SELECT id, user_id, target_name, commit_sha, branch, comment, state, created_at FROM deployments WHERE state = 'successful' AND application_name = ? AND target_name = ? AND created_at > ? ORDER BY created_at ASC;`
)

var ErrDeployInProgress = errors.New("another deployment to target already in progress")

func createDeployment(db *sql.DB, d *models.Deployment) error {
	var id int64
	var state models.DeploymentState = models.DEPLOYMENT_NEW
	var createdAt time.Time = time.Now()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	exists, err := activeDeploymentExists(tx, d.ApplicationName, d.TargetName)
	if err != nil {
		tx.Rollback()
		return err
	}
	if exists {
		tx.Rollback()
		return ErrDeployInProgress
	}

	result, err := tx.Exec(deploymentInsertStmt, d.UserId, d.ApplicationName,
		d.TargetName, d.CommitSha, d.Branch, d.Comment, string(state), createdAt)
	if err != nil {
		tx.Rollback()
		return err
	}

	id, err = result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	d.Id = int(id)
	d.State = state
	d.CreatedAt = createdAt

	return tx.Commit()
}

func updateDeploymentState(db *sql.DB, d *models.Deployment, state models.DeploymentState) error {
	_, err := db.Exec(deploymentUpdateStateStmt, string(state), d.Id)
	if err != nil {
		return err
	}

	d.State = state
	return nil
}

func getRecentApplicationDeployments(db *sql.DB, a *models.Application) ([]*models.Deployment, error) {
	return getApplicationDeployments(db, a, 10)
}

func getAllApplicationDeployments(db *sql.DB, a *models.Application) ([]*models.Deployment, error) {
	return getApplicationDeployments(db, a, -1)
}

func getApplicationDeployments(db *sql.DB, a *models.Application, limit int) ([]*models.Deployment, error) {
	rows, err := db.Query(applicationDeploymentsStmt, a.Name, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return readApplicationDeployments(rows)
}

func getApplicationDeploymentsByTarget(db *sql.DB, a *models.Application, t *models.Target) ([]*models.Deployment, error) {
	rows, err := db.Query(applicationDeploymentsByTargetStmt, a.Name, t.Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return readApplicationDeployments(rows)
}

func readApplicationDeployments(rows *sql.Rows) ([]*models.Deployment, error) {
	deployments := []*models.Deployment{}

	for rows.Next() {
		var state string
		d := &models.Deployment{}

		err := rows.Scan(&d.Id, &d.UserId, &d.TargetName, &d.CommitSha, &d.Branch, &d.Comment, &state, &d.CreatedAt)
		if err != nil {
			return deployments, err
		}

		d.State = models.DeploymentState(state)

		deployments = append(deployments, d)
	}

	if err := rows.Err(); err != nil {
		return deployments, err
	}

	return deployments, nil
}

func getDeployment(db *sql.DB, id int) (*models.Deployment, error) {
	return queryDeploymentRow(db, deploymentStmt, id)
}

func getLastTargetDeployment(db *sql.DB, a *models.Application, targetName string) (*models.Deployment, error) {
	return queryDeploymentRow(db, lastTargetDeploymentStmt,
		string(models.DEPLOYMENT_SUCCESSFUL), a.Name, targetName)
}

func getDailyDigestDeployments(db *sql.DB, a *models.Application, targetName string, since time.Time) ([]*models.Deployment, error) {
	deployments := []*models.Deployment{}

	rows, err := db.Query(dailyDigestDeploymentsStmt, a.Name, targetName, since)
	if err != nil {
		return deployments, err
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		d := &models.Deployment{}

		err = rows.Scan(&d.Id, &d.UserId, &d.TargetName, &d.CommitSha, &d.Branch, &d.Comment, &state, &d.CreatedAt)
		if err != nil {
			return deployments, err
		}

		d.State = models.DeploymentState(state)

		deployments = append(deployments, d)
	}

	if err := rows.Err(); err != nil {
		return deployments, err
	}

	return deployments, nil
}

func failUnfinishedDeployments(db *sql.DB) error {
	_, err := db.Exec(deploymentFailUnfinishedStmt,
		string(models.DEPLOYMENT_FAILED), string(models.DEPLOYMENT_NEW),
		string(models.DEPLOYMENT_ACTIVE))
	return err
}

func createLogEntry(db *sql.DB, entry *deploy.LogEntry) error {
	result, err := db.Exec(logEntryInsertStmt, entry.DeploymentId,
		string(entry.EntryType), entry.Origin, entry.Message,
		entry.Timestamp, time.Now())
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	entry.Id = int(id)

	return err
}

func getDeploymentLogEntries(db *sql.DB, d *models.Deployment) ([]*deploy.LogEntry, error) {
	entries := []*deploy.LogEntry{}

	rows, err := db.Query(deploymentLogEntriesStmt, d.Id)
	if err != nil {
		return entries, err
	}
	defer rows.Close()

	for rows.Next() {
		var entryType string
		e := &deploy.LogEntry{}

		err = rows.Scan(&e.Id, &e.DeploymentId, &entryType, &e.Origin, &e.Message, &e.Timestamp)
		if err != nil {
			return entries, err
		}

		e.EntryType = deploy.LogEntryType(entryType)

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return entries, err
	}

	return entries, nil
}

func newLogEntrySaver(db *sql.DB) deploy.Listener {
	fn := func(logs <-chan deploy.LogEntry) {
		for entry := range logs {
			err := createLogEntry(db, &entry)
			if err != nil {
				log.Printf("error saving log entry: %s", err)
			}
		}
	}

	return fn
}

func createUser(db *sql.DB, u *models.User) error {
	u.ApiToken = uuid.New()
	_, err := db.Exec(userInsertStmt, u.Id, u.Name, u.AccessToken, u.AvatarUrl, u.ApiToken)
	return err
}

func updateUser(db *sql.DB, u *models.User) error {
	_, err := db.Exec(userUpdateStmt, u.AccessToken, u.AvatarUrl, u.Id)
	return err
}

func getUser(db *sql.DB, id int) (*models.User, error) {
	u := &models.User{}

	err := db.QueryRow(userStmt, id).Scan(&u.Id, &u.Name, &u.AccessToken, &u.AvatarUrl, &u.ApiToken)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func getUserByApiToken(db *sql.DB, token string) (*models.User, error) {
	u := &models.User{}

	err := db.QueryRow(userApiTokenStmt, token).Scan(&u.Id, &u.Name, &u.AccessToken, &u.AvatarUrl, &u.ApiToken)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func getUsers(db *sql.DB, ids []int) ([]*models.User, error) {
	users := []*models.User{}

	if len(ids) == 0 {
		return users, nil
	}

	stmt := selectUsersStmt(ids)

	args := []interface{}{}
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := db.Query(stmt, args...)
	if err != nil {
		return users, err
	}
	defer rows.Close()

	for rows.Next() {
		u := &models.User{}

		err = rows.Scan(&u.Id, &u.Name, &u.AccessToken, &u.AvatarUrl)
		if err != nil {
			return users, err
		}

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return users, err
	}

	return users, nil
}

func createOrUpdateUser(db *sql.DB, u *models.User) error {
	saved, err := getUser(db, u.Id)
	if saved != nil && err == nil {
		err = updateUser(db, u)
		return err
	}
	err = createUser(db, u)
	return err
}

func loadDeploymentsUsers(db *sql.DB, deployments []*models.Deployment) error {
	// Set up map to have unique id->pointer mappings
	uniqueUserIds := map[int]*models.User{}
	for _, d := range deployments {
		uniqueUserIds[d.UserId] = nil
	}

	// Collect the now unique (keys in map are unique)
	userIds := []int{}
	for k := range uniqueUserIds {
		userIds = append(userIds, k)
	}

	users, err := getUsers(db, userIds)
	if err != nil {
		return err
	}

	// Set up the pointers to the users
	for _, u := range users {
		uniqueUserIds[u.Id] = u
	}

	// Go over deployments again, set pointers to correct user
	for _, d := range deployments {
		d.User = uniqueUserIds[d.UserId]
	}

	return nil
}

func selectUsersStmt(ids []int) string {
	tmpl := "SELECT id, name, access_token, avatar_url FROM users WHERE id IN (?"
	stmt := tmpl + strings.Repeat(",?", len(ids)-1) + ");"
	return stmt
}

func activeDeploymentExists(tx *sql.Tx, applicationName, targetName string) (bool, error) {
	var state string
	err := tx.QueryRow(activeDeploymentsStmt, applicationName, targetName).Scan(&state)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	default:
		return true, nil
	}
}

func queryDeploymentRow(db *sql.DB, query string, args ...interface{}) (*models.Deployment, error) {
	d := &models.Deployment{}
	var state string

	err := db.QueryRow(query, args...).Scan(&d.Id, &d.UserId, &d.ApplicationName,
		&d.TargetName, &d.CommitSha, &d.Branch, &d.Comment, &state, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.State = models.DeploymentState(state)

	return d, nil
}

func isMigrated(db *sql.DB) (bool, error) {
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
