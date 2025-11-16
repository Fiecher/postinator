package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"postinator/internal/bot"
	"postinator/internal/files"
	"postinator/internal/services"
	"postinator/internal/storage"

	"github.com/mymmrac/telego"
)

type PhotoHandler struct {
	imageService *services.ImageService
	bot          bot.Bot
	fileManager  files.FileManager
	photoStorage *storage.InMemoryPhotoStorage
	logger       *log.Logger
}

func NewPhotoHandler(
	imageService *services.ImageService,
	bot bot.Bot,
	fileManager files.FileManager,
	photoStorage *storage.InMemoryPhotoStorage,
	logger *log.Logger,
) *PhotoHandler {
	return &PhotoHandler{
		imageService: imageService,
		bot:          bot,
		fileManager:  fileManager,
		photoStorage: photoStorage,
		logger:       logger,
	}
}

func (ph *PhotoHandler) HandleUpdate(ctx context.Context, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	chatID := msg.Chat.ID

	if !hasPhoto(msg) {
		if err := ph.bot.SendText(ctx, chatID, "🔗 Please send a photo or document with caption (optional)."); err != nil {
			ph.logger.Println("Error sending text:", err)
		}
		return
	}

	err := ph.withProcessing(ctx, chatID, func() error {
		if err := ph.bot.SendText(ctx, chatID, "⏳ Postinating..."); err == nil {
			ph.logger.Println("Error sending text:", err)
		}

		resultPath, cleanup, err := ph.process(ctx, msg)
		if err != nil {
			return ph.fail(chatID, "processing failed", "🚧 Error processing image", err)
		}
		defer cleanup()

		if err := ph.bot.SendFileAuto(ctx, chatID, resultPath); err != nil {
			return ph.fail(chatID, "send error", "🚧 Error sending result", err)
		}

		if err = ph.bot.SendText(ctx, chatID, "✅️ Image postinated successfully!"); err != nil {
			ph.logger.Println("Error sending text:", err)
		}
		return nil
	})

	if err != nil {
		ph.logger.Printf("HandleUpdate failed: %v", err)
	}
}

func (ph *PhotoHandler) process(ctx context.Context, msg *telego.Message) (string, func(), error) {
	fileID, err := extractFileID(msg)
	if err != nil {
		return "", nil, err
	}

	// скачиваем в temp
	localPath, cleanupTemp, err := ph.fileManager.DownloadToTemp(ctx, fileID)
	if err != nil {
		return "", nil, fmt.Errorf("download failed: %w", err)
	}

	text := getText(msg)

	// рендерим
	resultPath, err := ph.imageService.Render(localPath, text)
	if err != nil {
		cleanupTemp()
		return "", nil, fmt.Errorf("render error: %w", err)
	}

	// cleanup для обоих файлов
	cleanup := func() {
		cleanupTemp()
		_ = os.Remove(resultPath)
	}

	return resultPath, cleanup, nil
}

func (ph *PhotoHandler) withProcessing(ctx context.Context, chatID int64, fn func() error) error {
	if ph.photoStorage.IsProcessing(chatID) {
		if err := ph.bot.SendText(ctx, chatID, "😵‍💫 Slow down, I'm already postinatin' it."); err != nil {
			return err
		}
		return fmt.Errorf("already processing")
	}

	ph.photoStorage.SetProcessing(chatID)
	defer ph.photoStorage.ClearProcessing(chatID)

	return fn()
}

func (ph *PhotoHandler) fail(chatID int64, logMsg, userMsg string, err error) error {
	ph.logger.Printf("%s: %v", logMsg, err)
	_ = ph.bot.SendText(context.Background(), chatID, userMsg)
	return err
}

func extractFileID(msg *telego.Message) (string, error) {
	switch {
	case len(msg.Photo) > 0:
		return msg.Photo[len(msg.Photo)-1].FileID, nil
	case msg.Document != nil:
		return msg.Document.FileID, nil
	default:
		return "", fmt.Errorf("no photo or document in message")
	}
}

func hasPhoto(msg *telego.Message) bool {
	return len(msg.Photo) > 0 || msg.Document != nil
}

func getText(msg *telego.Message) string {
	if msg.Caption != "" {
		return msg.Caption
	}
	return msg.Text
}
