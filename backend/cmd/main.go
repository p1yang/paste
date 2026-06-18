package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"paste/backend/internal/api"
	"paste/backend/internal/autostart"
	"paste/backend/internal/clipboard"
	"paste/backend/internal/config"
	"paste/backend/internal/logger"
	"paste/backend/internal/paste"
	"paste/backend/internal/security"
	"paste/backend/internal/storage"
)

var (
	port        = flag.Int("port", 48175, "API server port")
	dataDir     = flag.String("data-dir", "", "Data directory path")
	maxHistory  = flag.Int("max-history", 5000, "Maximum number of history items")
	showVersion = flag.Bool("version", false, "Show version")
)

const version = "1.0.0"

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("Paste Backend v%s\n", version)
		return
	}

	if *dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		*dataDir = home + "/Library/Application Support/Paste"
	}

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(*dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Sugar.Infow("starting paste backend",
		"version", version,
		"port", *port,
		"dataDir", *dataDir,
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Sugar.Errorw("panic recovered",
				"error", r,
				"stack", string(debug.Stack()),
			)
			os.Exit(1)
		}
	}()

	configMgr, err := config.NewManager(*dataDir)
	if err != nil {
		logger.Sugar.Fatalw("failed to initialize config manager", "error", err)
	}

	securityMgr := security.NewManager(configMgr.Get())

	storageMgr, err := storage.New(*dataDir, *maxHistory)
	if err != nil {
		logger.Sugar.Fatalw("failed to initialize storage", "error", err)
	}
	defer func() {
		if err := storageMgr.Close(); err != nil {
			logger.Sugar.Errorw("error closing storage", "error", err)
		}
	}()

	clipboardMonitor := clipboard.NewMonitor(storageMgr, securityMgr)
	if err := clipboardMonitor.Start(); err != nil {
		logger.Sugar.Fatalw("failed to start clipboard monitor", "error", err)
	}
	defer clipboardMonitor.Stop()

	pasteMgr := paste.NewManager(storageMgr, clipboardMonitor)

	autostartMgr := autostart.NewManager("Paste")
	if configMgr.Get().AutoStart {
		if err := autostartMgr.Enable(); err != nil {
			logger.Sugar.Warnw("failed to enable autostart", "error", err)
		}
	}

	server := api.NewServer(
		*port,
		storageMgr,
		clipboardMonitor,
		pasteMgr,
		configMgr,
		securityMgr,
		autostartMgr,
	)

	go func() {
		if err := server.Start(); err != nil {
			logger.Sugar.Fatalw("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	logger.Sugar.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		logger.Sugar.Errorw("error stopping server", "error", err)
	}

	logger.Sugar.Info("shutdown complete")
}
