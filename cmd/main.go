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
	"postinator/internal/services"
	"postinator/internal/storage"
)

func main() {
	logger := log.Default()
	cfg := config.Load(logger)

	photoStorage := storage.NewInMemoryPhotoStorage()
	imageService := services.NewImageService(cfg.AssetsDir, cfg.TempDir, cfg.MaxFileSize)
	botService, err := bot.NewTelegramBot(cfg.BotToken, logger, cfg.MaxFileSize)
	if err != nil {
		logger.Fatal(err)
	}

	fileManager, err := files.NewTelegramFileManager(botService, cfg.TempDir, cfg.BotToken)
	if err != nil {
		logger.Fatal(err)
	}

	photoHandler := handlers.NewPhotoHandler(imageService, botService, fileManager, photoStorage, logger)

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
