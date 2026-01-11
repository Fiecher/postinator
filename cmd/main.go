package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"postinator/internal/bot"
	"postinator/internal/config"
	"postinator/internal/files"
	"postinator/internal/handlers"
	"postinator/internal/image"
	"postinator/internal/services"
	"postinator/internal/storage"
)

func main() {
	logger := log.Default()
	cfg := config.Load(logger)

	assetLoader := files.NewAssetLoader(
		cfg.AssetsDir,
		cfg.BackgroundFile,
		cfg.BackgroundStatsFile,
		cfg.FontFile,
		cfg.OverlayFile,
	)

	botService, err := bot.NewTelegramBot(cfg.BotToken, logger, cfg.MaxFileSize)
	if err != nil {
		logger.Fatal(err)
	}

	processor := &image.Processor{}
	fileManager, err := files.NewTelegramFileManager(
		botService,
		cfg.TempDir,
		cfg.BotToken,
	)
	if err != nil {
		logger.Fatal(err)
	}

	imageService := services.NewImageService(
		assetLoader,
		processor,
		fileManager,
		cfg.TempDir,
	)

	photoStorage := storage.NewRenderStateStore()

	photoHandler := handlers.NewPhotoHandler(
		imageService,
		botService,
		fileManager,
		photoStorage,
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := botService.Start(ctx, photoHandler.HandleUpdate); err != nil {
			logger.Printf("Error starting bot: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
}
