package deploy

import (
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

var preDeployment = models.DeploymentStage("PRE_DEPLOYMENT")
var migrate = models.DeploymentStage("MIGRATE")

var testHosts []*models.Host = []*models.Host{
	{Name: "workers.applikatoni.com", Roles: []string{"workers"}},
	{Name: "webcluster.applikatoni.com", Roles: []string{"web", "migrator"}},
	{Name: "db.applikatoni.com", Roles: []string{"database"}},
}

var testRoles []*models.Role = []*models.Role{
	&models.Role{
		Name: "web",
		ScriptTemplates: map[models.DeploymentStage]string{
			preDeployment: "webfoobar",
		},
	},
	&models.Role{
		Name: "workers",
		ScriptTemplates: map[models.DeploymentStage]string{
			preDeployment: "workersfoobar",
		},
	},
	&models.Role{
		Name: "database",
		ScriptTemplates: map[models.DeploymentStage]string{
			preDeployment: "databasefoobar",
		},
	},
	&models.Role{
		Name: "migrator",
		ScriptTemplates: map[models.DeploymentStage]string{
			migrate: "migrating the database",
		},
	},
}

func TestFindHostRoles(t *testing.T) {
	for _, host := range testHosts {
		roles, err := findHostRoles(host, testRoles)
		if err != nil {
			t.Errorf("roles not found. err=%s", err)
		}

		if len(roles) != len(host.Roles) {
			t.Errorf("not enough roles found. expected=%d, got=%d", len(host.Roles), len(roles))
		}

		for _, expectedRole := range host.Roles {
			found := false
			for _, foundRole := range roles {
				if foundRole.Name == expectedRole {
					found = true
				}
			}
			if !found {
				t.Errorf("wrong roles for host found. want=%v, got=%v", host.Roles, roles)
			}
		}
	}
}

func TestFindHostRoleError(t *testing.T) {
	wrongHost := &models.Host{Name: "foobar.com", Roles: []string{"dancer"}}
	_, err := findHostRoles(wrongHost, testRoles)
	if err == nil {
		t.Errorf("error expected. but error is nil")
	}
}

var newWorkerTests = []struct {
	roles           []*models.Role
	host            *models.Host
	scriptOptions   map[string]string
	expectedScripts map[models.DeploymentStage]string
}{
	{ // Host has one role, one role available
		roles: []*models.Role{
			&models.Role{
				Name: "web",
				ScriptTemplates: map[models.DeploymentStage]string{
					preDeployment: "{{.foo}}{{.RubyVersion}}",
				},
				Options: map[string]string{"foo": "bar"},
			},
		},
		host: &models.Host{
			Name:  "webcluster.applikatoni.com",
			Roles: []string{"web"},
		},
		scriptOptions: map[string]string{"RubyVersion": "2.2.0"},
		expectedScripts: map[models.DeploymentStage]string{
			preDeployment: "bar2.2.0",
		},
	},
	{ // Host has one role, two roles available
		roles: []*models.Role{
			&models.Role{
				Name: "web",
				ScriptTemplates: map[models.DeploymentStage]string{
					preDeployment: "{{.foo}}{{.RubyVersion}}",
				},
				Options: map[string]string{"foo": "bar"},
			},
			&models.Role{
				Name: "unused",
				ScriptTemplates: map[models.DeploymentStage]string{
					preDeployment: "unused",
				},
				Options: map[string]string{"foo": "bar"},
			},
		},
		host: &models.Host{
			Name:  "webcluster.applikatoni.com",
			Roles: []string{"web"},
		},
		scriptOptions: map[string]string{"RubyVersion": "2.2.0"},
		expectedScripts: map[models.DeploymentStage]string{
			preDeployment: "bar2.2.0",
		},
	},
	{ // Host has two roles, two matching roles available
		roles: []*models.Role{
			&models.Role{
				Name: "web",
				ScriptTemplates: map[models.DeploymentStage]string{
					preDeployment: "predeployment",
				},
				Options: map[string]string{"foo": "bar"},
			},
			&models.Role{
				Name: "migrator",
				ScriptTemplates: map[models.DeploymentStage]string{
					migrate: "migrate",
				},
				Options: map[string]string{"foo": "bar"},
			},
		},
		host: &models.Host{
			Name:  "webcluster.applikatoni.com",
			Roles: []string{"web", "migrator"},
		},
		scriptOptions: map[string]string{"RubyVersion": "2.2.0"},
		expectedScripts: map[models.DeploymentStage]string{
			preDeployment: "predeployment",
			migrate:       "migrate",
		},
	},
	{ // Host has two roles, three roles available
		roles: []*models.Role{
			&models.Role{
				Name: "web",
				ScriptTemplates: map[models.DeploymentStage]string{
					preDeployment: "predeployment",
				},
				Options: map[string]string{"foo": "bar"},
			},
			&models.Role{
				Name: "migrator",
				ScriptTemplates: map[models.DeploymentStage]string{
					migrate: "migrate",
				},
				Options: map[string]string{"foo": "bar"},
			},
			&models.Role{
				Name: "database",
				ScriptTemplates: map[models.DeploymentStage]string{
					preDeployment: "database",
				},
				Options: map[string]string{"foo": "bar"},
			},
		},
		host: &models.Host{
			Name:  "webcluster.applikatoni.com",
			Roles: []string{"web", "migrator"},
		},
		scriptOptions: map[string]string{"RubyVersion": "2.2.0"},
		expectedScripts: map[models.DeploymentStage]string{
			preDeployment: "predeployment",
			migrate:       "migrate",
		},
	},
}

func TestNewWorker(t *testing.T) {
	testLogger := &DeploymentLogger{}
	testSshConfig, _ := newSSHClientConfig("testuser", []byte("testsshkey"))
	testManager := &Manager{logger: testLogger, sshConfig: testSshConfig}

	for _, tt := range newWorkerTests {
		testConfig := &models.DeploymentConfig{Roles: tt.roles}
		testManager.config = testConfig

		w, err := testManager.newWorker(tt.host, tt.scriptOptions)
		if err != nil {
			t.Error(err)
		}
		if len(tt.expectedScripts) != len(w.scripts) {
			t.Errorf("worker has wrong number of scripts. want=%d, got=%d", len(tt.expectedScripts), len(w.scripts))
		}
		for k, v := range tt.expectedScripts {
			workerScript, ok := w.scripts[k]
			if !ok {
				t.Errorf("worker has no script for %s", k)
			}
			if workerScript != v {
				t.Errorf("worker has wrong script. want=%q, got=%q", v, workerScript)
			}
		}
		if w.logger != testLogger {
			t.Errorf("worker has the wrong logger. want=%+v, got=%+v", testLogger, w.logger)
		}
		if w.sshConfig != testSshConfig {
			t.Errorf("worker has the wrong sshConfig. want=%+v, got=%+v", testSshConfig, w.sshConfig)
		}
	}
}

func TestNewWorkerError(t *testing.T) {
	testLogger := &DeploymentLogger{}
	testSshConfig, _ := newSSHClientConfig("testuser", []byte("testsshkey"))
	testManager := &Manager{logger: testLogger, sshConfig: testSshConfig}

	// Two roles that define scripts for the same stage
	roles := []*models.Role{
		&models.Role{
			Name: "web",
			ScriptTemplates: map[models.DeploymentStage]string{
				preDeployment: "predeployment",
			},
			Options: map[string]string{},
		},
		&models.Role{
			Name: "workers",
			ScriptTemplates: map[models.DeploymentStage]string{
				preDeployment: "predeployment",
			},
			Options: map[string]string{},
		},
	}

	host := &models.Host{
		Name:  "webcluster.applikatoni.com",
		Roles: []string{"web", "workers"},
	}

	scriptOptions := map[string]string{"RubyVersion": "2.2.0"}

	testConfig := &models.DeploymentConfig{Roles: roles}
	testManager.config = testConfig

	w, err := testManager.newWorker(host, scriptOptions)
	if err == nil {
		t.Errorf("newWorker expected to return error, but didn't")
	}
	if w != nil {
		t.Errorf("newWorker expected to not return a new worker, but did")
	}
}
