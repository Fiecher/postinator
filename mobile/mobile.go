package mobile

import (
	"context"
	"fmt"
	"log"
	"postinator/internal/bot"
	"postinator/internal/files"
	"postinator/internal/handlers"
	"postinator/internal/image"
	"postinator/internal/services"
)

type BotControl struct {
	cancel context.CancelFunc
}

func NewBotControl() *BotControl {
	return &BotControl{}
}

func (bc *BotControl) StartBot(token string, assetsDir string, tempDir string) string {
	if bc.cancel != nil {
		return "Bot already started"
	}

	logger := log.Default()

	assetLoader := files.NewAssetLoader(
		assetsDir,
		"BG1.png",
		"BG2.png",
		"font.ttf",
		"Overlay1.png",
	)

	botService, err := bot.NewTelegramBot(token, logger, 50*1024*1024)
	if err != nil {
		return fmt.Sprintf("Error creating bot: %v", err)
	}

	fileManager, err := files.NewTelegramFileManager(
		botService,
		tempDir,
		token,
	)
	if err != nil {
		return fmt.Sprintf("Error creating file manager: %v", err)
	}

	imageService := services.NewImageService(
		assetLoader,
		fileManager,
		tempDir,
	)

	photoStorage := image.NewRenderStateStore()
	photoHandler := handlers.NewHandler(
		imageService,
		botService,
		fileManager,
		photoStorage,
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())
	bc.cancel = cancel

	go func() {
		log.Println("Bot goroutine started")
		if err := botService.Start(ctx, photoHandler.HandleUpdate); err != nil {
			log.Printf("Error starting bot: %v", err)
			bc.cancel = nil
		}
	}()

	return "Bot started successfully"
}

func (bc *BotControl) StopBot() {
	if bc.cancel != nil {
		bc.cancel()
		bc.cancel = nil
		log.Println("Bot stopped by user")
	}
}
