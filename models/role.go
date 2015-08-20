package models

import "bytes"
import "text/template"

type DeploymentStage string

type Role struct {
	Name            string                     `json:"name"`
	ScriptTemplates map[DeploymentStage]string `json:"script_templates"`
	Options         map[string]string          `json:"options"`
}

func (r *Role) RenderScripts(options map[string]string) (map[DeploymentStage]string, error) {
	rendered := make(map[DeploymentStage]string)
	mergedOptions := mergeOptions(r.Options, options)

	for stage, scriptTemplate := range r.ScriptTemplates {
		var b bytes.Buffer

		tmpl, err := template.New(string(stage)).Parse(scriptTemplate)
		if err != nil {
			return nil, err
		}

		err = tmpl.Execute(&b, mergedOptions)
		if err != nil {
			return nil, err
		}

		rendered[stage] = b.String()
	}

	return rendered, nil
}

func mergeOptions(o1 map[string]string, o2 map[string]string) map[string]string {
	for key, value := range o2 {
		o1[key] = value
	}
	return o1
}
