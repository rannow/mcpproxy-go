//go:build nogui || headless || linux

package tray

import (
	"context"

	"go.uber.org/zap"
)

// ServerInterface is now defined in interfaces.go (shared across all build configurations)
// In headless builds, the stub uses the common ServerInterface from interfaces.go

// App represents the system tray application (stub version)
type App struct {
	logger *zap.SugaredLogger
}

// New creates a new tray application (stub version)
func New(_ ServerInterface, logger *zap.SugaredLogger, _ string, _ string, _ func()) *App {
	return &App{
		logger: logger,
	}
}

// Run starts the system tray application (stub version - does nothing)
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("Tray functionality disabled (nogui/headless build)")
	<-ctx.Done()
	return ctx.Err()
}
