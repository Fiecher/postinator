package mobile

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"postinator/internal/bot"
	"postinator/internal/config"
	"postinator/internal/files"
	"postinator/internal/handlers"
	"postinator/internal/image"
	"postinator/internal/services"
	"postinator/internal/toggl"
)

type BotControl struct {
	cancel context.CancelFunc
}

func NewBotControl() *BotControl {
	return &BotControl{}
}

func (bc *BotControl) StartBot(configDir string) string {
	if bc.cancel != nil {
		return "Bot already started"
	}

	logger := log.Default()

	cfg, err := loadConfigFromPath(configDir)
	if err != nil {
		return fmt.Sprintf("Config error: %v", err)
	}

	if !filepath.IsAbs(cfg.AssetsDir) {
		cfg.AssetsDir = filepath.Join(configDir, cfg.AssetsDir)
	}
	if !filepath.IsAbs(cfg.TempDir) {
		cfg.TempDir = filepath.Join(configDir, cfg.TempDir)
	}

	assetLoader := files.NewAssetLoader(
		cfg.AssetsDir,
		cfg.BackgroundFile,
		cfg.BackgroundStatsFile,
		cfg.FontFile,
		cfg.OverlayFile,
	)

	botService, err := bot.NewTelegramBot(cfg.BotToken, logger, cfg.MaxFileSize)
	if err != nil {
		return fmt.Sprintf("Error creating bot: %v", err)
	}

	fileManager, err := files.NewTelegramFileManager(
		botService,
		cfg.TempDir,
		cfg.BotToken,
	)
	if err != nil {
		return fmt.Sprintf("Error creating file manager: %v", err)
	}

	imageService := services.NewImageService(
		assetLoader,
		fileManager,
		cfg.TempDir,
	)

	togglClient := toggl.NewClient(cfg.TogglToken, cfg.TogglWorkspaceID)
	togglService := services.NewTogglService(togglClient, cfg.Stats)

	photoStorage := image.NewRenderStateStore()

	photoHandler := handlers.NewHandler(
		imageService,
		togglService,
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

func loadConfigFromPath(dir string) (*config.Config, error) {
	configPath := filepath.Join(dir, "config.yaml")
	return config.LoadFromPath(configPath)
}
