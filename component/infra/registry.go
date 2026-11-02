// Package infra contains the service-specific infrastructure handlers and
// the registry used by the CLI to find them.
package infra

import (
	"context"
	"fmt"
	"sort"

	"orb/component"
	"orb/executor"
)

// Lifecycle is implemented by each infrastructure component. The operations
// are shared at the command boundary, while each component owns its behavior.
type Lifecycle interface {
	Install(context.Context, component.DeploymentConfig, executor.Runner) error
	Upgrade(context.Context, component.DeploymentConfig, executor.Runner) error
	Status(context.Context, component.DeploymentConfig, executor.Runner) error
	Start(context.Context, component.DeploymentConfig, executor.Runner) error
	Stop(context.Context, component.DeploymentConfig, executor.Runner) error
	Restart(context.Context, component.DeploymentConfig, executor.Runner) error
}

// Registry maps the CLI's --service names to component implementations.
type Registry struct {
	components map[string]Lifecycle
}

func NewRegistry() *Registry {
	return &Registry{components: map[string]Lifecycle{
		"postgresql": PostgreSQL{},
	}}
}

// Names returns the registered service names in stable, alphabetical order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.components))
	for name := range r.components {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SetupComponents exposes registered components that participate in the
// common setup pipeline. A component without setup handlers is a programming
// error because setup is expected to configure every registered component.
func (r *Registry) SetupComponents() ([]component.SetupComponent, error) {
	components := make([]component.SetupComponent, 0, len(r.components))
	for _, name := range r.Names() {
		setupComponent, ok := r.components[name].(component.SetupComponent)
		if !ok {
			return nil, fmt.Errorf("infrastructure component %q does not implement setup handlers", name)
		}
		components = append(components, setupComponent)
	}
	return components, nil
}

func (r *Registry) Run(ctx context.Context, name, operation string, deployment component.DeploymentConfig, runner executor.Runner) error {
	if name != "" {
		component, ok := r.components[name]
		if !ok {
			return fmt.Errorf("unknown infrastructure service %q (available: %v)", name, r.Names())
		}
		return run(ctx, name, operation, component, deployment, runner)
	}

	for _, componentName := range r.Names() {
		if err := run(ctx, componentName, operation, r.components[componentName], deployment, runner); err != nil {
			return err
		}
	}
	return nil
}

func run(ctx context.Context, name, operation string, current Lifecycle, deployment component.DeploymentConfig, runner executor.Runner) error {
	var err error
	switch operation {
	case "install":
		err = current.Install(ctx, deployment, runner)
	case "upgrade":
		err = current.Upgrade(ctx, deployment, runner)
	case "status":
		err = current.Status(ctx, deployment, runner)
	case "start":
		err = current.Start(ctx, deployment, runner)
	case "stop":
		err = current.Stop(ctx, deployment, runner)
	case "restart":
		err = current.Restart(ctx, deployment, runner)
	default:
		return fmt.Errorf("unsupported lifecycle operation %q", operation)
	}
	if err != nil {
		return fmt.Errorf("%s %s: %w", operation, name, err)
	}
	return nil
}
