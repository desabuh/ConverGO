package cluster

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/desabuh/convergo/log"
	"github.com/desabuh/convergo/p2p"
)

var (
	MODULE_ALREADY_SHUT_DOWN = errors.New("AppNodeModule already shut down")
)

type AppNodeModule[M comparable, T comparable, R comparable] struct {
	Id T
	p2p.NetworkModule[M, T, R]
	log.GlobalLogger

	moduleShutdown *atomic.Bool
}

func (a *AppNodeModule[M, T, R]) GetId() T {
	return a.Id
}

func (a *AppNodeModule[M, T, R]) IsActive() bool {
	return !a.moduleShutdown.Load()
}

func (a *AppNodeModule[M, T, R]) InitModule(ctx context.Context) {
	err := a.Transport.ListenFor(a.Id)
	a.Log("Transport interface closed with error: %v", err)
}

func (a *AppNodeModule[M, T, R]) ShutDownModule() error {
	if !a.IsActive() {
		return MODULE_ALREADY_SHUT_DOWN
	}

	a.moduleShutdown.Store(true)

	a.Log("Shutting down application module...")
	a.MessageBroker.Stop()
	return a.Transport.Shutdown()
}

type AppNodeModuleFactory[M comparable, T interface {
	Format() string
	comparable
}, R comparable] struct {
	NetworkModuleFactory p2p.NetworkModuleFactory[M, T, R]
	LoggerFactory        log.GlobalLoggerFactory
}

func (a AppNodeModuleFactory[M, T, R]) Create(info T) AppNodeModule[M, T, R] {
	networkModule := a.NetworkModuleFactory.CreateFromArgs(info, map[string]any{})
	logger := a.LoggerFactory.CreateFromArgs(info.Format(), map[string]any{})

	return AppNodeModule[M, T, R]{
		Id:             info,
		NetworkModule:  networkModule,
		GlobalLogger:   logger,
		moduleShutdown: new(atomic.Bool),
	}
}

func (a AppNodeModuleFactory[M, T, R]) CreateFromArgs(info T, args map[string]any) AppNodeModule[M, T, R] {

	networkModule := a.NetworkModuleFactory.CreateFromArgs(info, args)
	logger := a.LoggerFactory.CreateFromArgs(info.Format(), args)

	return AppNodeModule[M, T, R]{
		Id:             info,
		NetworkModule:  networkModule,
		GlobalLogger:   logger,
		moduleShutdown: new(atomic.Bool),
	}

}
