package component

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"text/template"

	composeTemplates "orb/deployment/compose"
	"orb/executor"
)

// Prompter lets component setup logic request input without depending on a
// particular terminal UI implementation.
type Prompter interface {
	Text(context.Context, string) (string, error)
	Secret(context.Context, string) (string, error)
}

// SetupComponent participates in the shared deployment setup pipeline. Each
// component gathers its own inputs and prepares its own deployment artifacts.
type SetupComponent interface {
	Name() string
	GatherInputs(context.Context, *DeploymentConfig, Prompter) error
	Setup(context.Context, DeploymentConfig, executor.Runner) error
}

// CreateCompose renders a Compose template with component-specific values.
// The template stays in deployment/compose while shared rendering behavior
// lives with the component setup helpers.
func CreateCompose(templateName string, values any) ([]byte, error) {
	contents, err := composeTemplates.Template(templateName)
	if err != nil {
		return nil, fmt.Errorf("read Compose template %q: %w", templateName, err)
	}
	tmpl, err := template.New(templateName).Funcs(template.FuncMap{
		"quote": strconv.Quote,
	}).Option("missingkey=error").Parse(string(contents))
	if err != nil {
		return nil, fmt.Errorf("parse Compose template %q: %w", templateName, err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, values); err != nil {
		return nil, fmt.Errorf("render Compose template %q: %w", templateName, err)
	}
	return rendered.Bytes(), nil
}

// GatherInputs runs the input phase for each registered setup component.
func GatherInputs(ctx context.Context, components []SetupComponent, deployment *DeploymentConfig, prompter Prompter) error {
	for _, current := range components {
		if err := current.GatherInputs(ctx, deployment, prompter); err != nil {
			return fmt.Errorf("gather %s inputs: %w", current.Name(), err)
		}
	}
	return nil
}

// Prepare runs the artifact and target preparation phase for each component.
func Prepare(ctx context.Context, components []SetupComponent, deployment DeploymentConfig, runner executor.Runner) error {
	for _, current := range components {
		if err := current.Setup(ctx, deployment, runner); err != nil {
			return fmt.Errorf("prepare %s: %w", current.Name(), err)
		}
	}
	return nil
}
