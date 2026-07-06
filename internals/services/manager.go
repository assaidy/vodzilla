package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

type Manager struct {
	logger          *slog.Logger
	services        map[string]Service
	startedServices map[string]Service
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		logger:          logger,
		services:        make(map[string]Service),
		startedServices: make(map[string]Service),
	}
}

func (me *Manager) Add(name string, service Service) {
	if _, ok := me.services[name]; ok {
		panic(fmt.Sprintf("double registration of service: %s", name))
	}
	me.services[name] = service
}

func (me *Manager) StartAll(timeout ...time.Duration) {
	t := 10 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}

	me.logger.Info("starting all services", "pid", os.Getpid())
	defer me.logger.Info("started all services successfully", "pid", os.Getpid())

	for name, service := range me.services {
		me.logger.Info("starting service", "name", name, "pid", os.Getpid())

		ctx, cancel := context.WithTimeout(context.Background(), t)
		if err := service.Start(ctx); err != nil {
			me.logger.Error("failed to start service", "name", name, "pid", os.Getpid())
		} else {
			me.startedServices[name] = service
		}
		cancel()
	}
}

func (me *Manager) StopAll(timeout ...time.Duration) {
	t := 10 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}

	me.logger.Info("stopping all services", "pid", os.Getpid())
	defer me.logger.Info("stopped all services successfully", "pid", os.Getpid())

	for name, service := range me.startedServices {
		me.logger.Info("stopping service", "name", name, "pid", os.Getpid())

		ctx, cancel := context.WithTimeout(context.Background(), t)
		if err := service.Stop(ctx); err != nil {
			me.logger.Error("failed to stop service", "name", name, "pid", os.Getpid())
		}
		cancel()
	}
}
