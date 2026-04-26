package main

import (
	"context"
	"kerbecs/admin"
	"kerbecs/config"
	"kerbecs/gateway"
	"kerbecs/pkg/logger"
	"kerbecs/provider"
	"kerbecs/router"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

func main() {
	logger.Init(os.Getenv("ENV") == "PROD")
	defer logger.Logger.Sync()

	path := config.FilePath()
	file, err := config.LoadFile(path)
	if err != nil {
		logger.SugarLogger.Fatalf("Failed to load config %s: %v", path, err)
	}
	for _, w := range config.ApplyDefaults(file) {
		logger.SugarLogger.Warnln(w)
	}
	logger.SugarLogger.Infof("Loaded config from %s", path)

	config.PrintStartupBanner(file.Gateway.Env)

	state, err := buildLiveState(file)
	if err != nil {
		logger.SugarLogger.Fatalf("Failed to build initial state: %v", err)
	}
	statePtr := &gateway.StatePointer{}
	statePtr.Store(state)

	handlerCfg := gateway.HandlerConfig{
		GatewayName:    firstNonEmpty(file.Gateway.Name, config.Name),
		GatewayVersion: firstNonEmpty(file.Gateway.Version, config.Version),
	}
	listenerCfg := gateway.ListenerConfig{
		Port: file.Listeners.Gateway.Port,
		Env:  file.Gateway.Env,
		CORS: file.Listeners.Gateway.CORS,
	}
	adminCfg := admin.Config{
		Port:     file.Listeners.Admin.Port,
		Env:      file.Gateway.Env,
		Username: file.Listeners.Admin.Auth.Username,
		Password: file.Listeners.Admin.Auth.Password,
		CORS:     file.Listeners.Admin.CORS,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Optional config-file watcher: when providers.static.watch is true,
	// re-read the YAML on changes and atomically swap the live state.
	if file.Providers.Static.Watch {
		w := config.NewWatcher(path, func() { reload(path, statePtr) })
		go func() {
			if err := w.Start(ctx); err != nil {
				logger.SugarLogger.Errorf("config watcher: %v", err)
			}
		}()
		logger.SugarLogger.Infof("watching %s for changes", path)
	}

	var eg errgroup.Group
	eg.Go(func() error { return admin.Serve(ctx, adminCfg) })
	eg.Go(func() error { return gateway.Serve(ctx, listenerCfg, handlerCfg, statePtr) })

	if err := eg.Wait(); err != nil {
		logger.SugarLogger.Fatalf("server error: %v", err)
	}
	logger.SugarLogger.Infoln("shutdown complete")
}

// buildLiveState reads the parsed config and constructs the runtime state
// (provider, router, transport cache) that the proxy handler consumes.
func buildLiveState(file *config.File) (*gateway.LiveState, error) {
	static, err := provider.NewStatic(file)
	if err != nil {
		return nil, err
	}
	rt, err := router.New(static)
	if err != nil {
		return nil, err
	}
	logger.SugarLogger.Infof("Static provider loaded %d route(s)", len(static.Routes()))
	return gateway.BuildState(rt), nil
}

// reload re-reads the config file and atomically swaps the live state.
// On any failure, the existing state stays in place — bad config never
// brings the gateway down.
func reload(path string, statePtr *gateway.StatePointer) {
	logger.SugarLogger.Infof("config change detected, reloading %s", path)
	file, err := config.LoadFile(path)
	if err != nil {
		logger.SugarLogger.Errorf("reload: parse failed, keeping previous config: %v", err)
		return
	}
	for _, w := range config.ApplyDefaults(file) {
		logger.SugarLogger.Warnln(w)
	}
	newState, err := buildLiveState(file)
	if err != nil {
		logger.SugarLogger.Errorf("reload: build failed, keeping previous config: %v", err)
		return
	}
	statePtr.Store(newState)
	logger.SugarLogger.Infoln("config reload complete")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
