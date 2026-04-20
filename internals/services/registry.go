package services

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
)

type Registry struct {
	services        map[string]Service
	startedServices map[string]Service
	logger          *log.Logger
}

func NewRegistry(logger *log.Logger) *Registry {
	return &Registry{
		services:        make(map[string]Service),
		startedServices: make(map[string]Service),
		logger:          logger,
	}
}

func (me *Registry) Add(name string, s Service) {
	if _, ok := me.services[name]; ok {
		panic(fmt.Sprintf("double registration of service: %s", name))
	}
	me.services[name] = s
}

func (me *Registry) Start(ctx context.Context) {
	me.logger.Info("starting all services...")
	defer me.logger.Info("started all services successfully")

	for name, service := range me.services {
		me.logger.Info("starting service", "name", name)
		if err := service.Start(ctx); err != nil {
			me.logger.Error("failed to start service", "name", name)
		} else {
			me.startedServices[name] = service
		}
	}
}

func (me *Registry) Stop(ctx context.Context) {
	me.logger.Info("stopping all services...")
	defer me.logger.Info("stopped all services successfully")

	for name, service := range me.startedServices {
		me.logger.Info("stopping service", "name", name)
		if err := service.Stop(ctx); err != nil {
			me.logger.Error("failed to stop service", "name", name)
		}
	}
}
