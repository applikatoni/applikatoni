package models

import (
	"fmt"
	"testing"
	"time"
)

var otherOptions = map[string]string{
	"CommitSha":       "FAKESHA",
	"AssetsTimestamp": time.Now().Format(assetsTimestampLayout),
	"AnotherOption":   "AnotherValue",
}

var renderTests = []struct {
	role           Role
	templates      map[DeploymentStage]string
	roleOptions    map[string]string
	optionsToMerge map[string]string
	expectation    map[DeploymentStage]string
}{
	{
		templates: map[DeploymentStage]string{
			DeploymentStage("PRE_DEPLOYMENT"): "echo {{.Dir}} {{.RubyVersion}} {{.RailsEnv}}",
		},
		roleOptions:    map[string]string{"Dir": "/home/foobar", "RubyVersion": "2.1.4", "RailsEnv": "staging"},
		optionsToMerge: map[string]string{},
		expectation: map[DeploymentStage]string{
			DeploymentStage("PRE_DEPLOYMENT"): `echo /home/foobar 2.1.4 staging`,
		},
	},
	{
		templates: map[DeploymentStage]string{
			DeploymentStage("CODE_DEPLOYMENT"): "echo {{.CommitSha}} {{.AssetsTimestamp}} {{.AnotherOption}}",
		},
		roleOptions:    map[string]string{},
		optionsToMerge: otherOptions,
		expectation: map[DeploymentStage]string{
			DeploymentStage("CODE_DEPLOYMENT"): fmt.Sprintf("echo %s %s %s", otherOptions["CommitSha"], otherOptions["AssetsTimestamp"], otherOptions["AnotherOption"]),
		},
	},
	{
		templates: map[DeploymentStage]string{
			DeploymentStage("CODE_DEPLOYMENT"): "echo NOVARIABLE",
		},
		roleOptions:    map[string]string{},
		optionsToMerge: otherOptions,
		expectation: map[DeploymentStage]string{
			DeploymentStage("CODE_DEPLOYMENT"): "echo NOVARIABLE",
		},
	},
}

func TestRender(t *testing.T) {
	for _, tt := range renderTests {
		role := &Role{ScriptTemplates: tt.templates, Options: tt.roleOptions}
		result, err := role.RenderScripts(tt.optionsToMerge)
		if err != nil {
			t.Errorf("RenderScripts returned error. err=%s", err)
		}

		for stage, expectedScript := range tt.expectation {
			if result[stage] != expectedScript {
				t.Errorf("Rendering wrong. expected='%s', got='%s'", expectedScript, result[stage])
			}
		}

		for name, _ := range otherOptions {
			if role.Options[name] != "" {
				t.Errorf("role.Options[%s] to be nil, got=%s", name, role.Options[name])
			}
		}
	}
}
