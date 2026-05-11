package registry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/assaidy/vodzilla/internals/services"
	"github.com/gofiber/fiber/v3"
)

type Registry struct {
	logger          *slog.Logger
	app             *fiber.App
	services        map[string]services.Service
	startedServices map[string]services.Service
}

func NewRegistry(logger *slog.Logger, app *fiber.App) *Registry {
	return &Registry{
		logger:          logger,
		app:             app,
		services:        make(map[string]services.Service),
		startedServices: make(map[string]services.Service),
	}
}

func (me *Registry) AddService(name string, service services.Service) {
	if _, ok := me.services[name]; ok {
		panic(fmt.Sprintf("double registration of service: %s", name))
	}
	me.services[name] = service
}

func (me *Registry) AddServiceWithInjection(name string, service services.Service) {
	me.AddService(name, service)
	me.Inject(name, service)
}

func (me *Registry) Inject(name string, dependency any) {
	me.app.State().Set(name, dependency)
}

func (me *Registry) Start(ctx context.Context) {
	me.logger.Info("starting all services", "pid", os.Getpid())
	defer me.logger.Info("started all services successfully", "pid", os.Getpid())

	for name, service := range me.services {
		me.logger.Info("starting service", "name", name, "pid", os.Getpid())
		if err := service.Start(ctx); err != nil {
			me.logger.Error("failed to start service", "name", name, "pid", os.Getpid())
		} else {
			me.startedServices[name] = service
		}
	}
}

func (me *Registry) Stop(ctx context.Context) {
	me.logger.Info("stopping all services", "pid", os.Getpid())
	defer me.logger.Info("stopped all services successfully", "pid", os.Getpid())

	for name, service := range me.startedServices {
		me.logger.Info("stopping service", "name", name, "pid", os.Getpid())
		if err := service.Stop(ctx); err != nil {
			me.logger.Error("failed to stop service", "name", name, "pid", os.Getpid())
		}
	}
}
