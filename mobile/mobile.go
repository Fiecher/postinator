package mobile

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"postinator/internal/bot"
	"postinator/internal/config"
	"postinator/internal/files"
	"postinator/internal/handlers"
	"postinator/internal/image"
	"postinator/internal/services"
	"postinator/internal/toggl"
)

type Logger interface {
	Log(msg string)
}

type BotControl struct {
	cancel context.CancelFunc
	logMu  sync.Mutex
	logger Logger
	logged bool
}

func NewBotControl() *BotControl {
	return &BotControl{}
}

func (bc *BotControl) SetLogger(l Logger) {
	bc.logMu.Lock()
	bc.logger = l
	bc.logMu.Unlock()
}

func (bc *BotControl) sendLog(s string) {
	bc.logMu.Lock()
	l := bc.logger
	bc.logMu.Unlock()
	if l != nil {
		go func() {
			defer func() {
				_ = recover()
			}()
			l.Log(s)
		}()
	}
}

type bcWriter struct {
	bc *BotControl
}

func (w *bcWriter) Write(p []byte) (int, error) {
	if w.bc != nil {
		w.bc.sendLog(string(p))
	}
	return len(p), nil
}

func (bc *BotControl) StartBot(configDir string) string {
	if bc.cancel != nil {
		return "Bot already started"
	}

	logger := log.New(&bcWriter{bc: bc}, "", log.LstdFlags)

	cfg, err := loadConfigFromPath(configDir)
	if err != nil {
		logger.Printf("Config error: %v", err)
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
		logger.Printf("Error creating bot: %v", err)
		return fmt.Sprintf("Error creating bot: %v", err)
	}

	fileManager, err := files.NewTelegramFileManager(
		botService,
		cfg.TempDir,
		cfg.BotToken,
	)
	if err != nil {
		logger.Printf("Error creating file manager: %v", err)
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
		logger.Println("Bot goroutine started")
		if err := botService.Start(ctx, photoHandler.HandleUpdate); err != nil {
			logger.Printf("Error starting bot: %v", err)
			bc.cancel = nil
		}
	}()

	return "Bot started successfully"
}

func (bc *BotControl) StopBot() {
	if bc.cancel != nil {
		bc.cancel()
		bc.cancel = nil
		bc.sendLog("Bot stopped by user")
	}
}

func loadConfigFromPath(dir string) (*config.Config, error) {
	configPath := filepath.Join(dir, "config.yaml")
	return config.LoadFromPath(configPath)
}

func (bc *BotControl) ReadConfigFile(configDir string) (string, error) {
	path := filepath.Join(configDir, "config.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (bc *BotControl) SaveConfigFile(configDir, contents string) error {
	path := filepath.Join(configDir, "config.yaml")
	return os.WriteFile(path, []byte(contents), 0644)
}

func (bc *BotControl) ValidateConfigString(configDir, contents string) error {
	tmp := filepath.Join(configDir, "config_validate_tmp.yaml")
	if err := os.WriteFile(tmp, []byte(contents), 0644); err != nil {
		return err
	}
	defer os.Remove(tmp)
	_, err := config.LoadFromPath(tmp)
	return err
}
