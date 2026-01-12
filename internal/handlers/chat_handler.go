package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"postinator/internal/bot"
	"postinator/internal/files"
	"postinator/internal/image"
	"postinator/internal/services"
	"strings"

	"github.com/mymmrac/telego"
)

type Handler struct {
	imageService *services.ImageService
	togglService *services.TogglService
	bot          bot.Bot
	fileManager  files.FileManager
	stateStore   *image.RenderStateStore
	logger       *log.Logger
}

func NewHandler(
	imageService *services.ImageService,
	togglService *services.TogglService,
	bot bot.Bot,
	fileManager files.FileManager,
	stateStore *image.RenderStateStore,
	logger *log.Logger,
) *Handler {
	return &Handler{
		imageService: imageService,
		togglService: togglService,
		bot:          bot,
		fileManager:  fileManager,
		stateStore:   stateStore,
		logger:       logger,
	}
}

func (ph *Handler) HandleUpdate(ctx context.Context, update telego.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	chatID := msg.Chat.ID

	switch msg.Text {
	case "/start":
		ph.stateStore.Finish(chatID)
		_ = ph.bot.ShowMenu(ctx, chatID)
		return
	case "🎫 Monthly-post":
		ph.stateStore.SetMode(chatID, image.ModeStats)
		_ = ph.bot.SendText(ctx, chatID, "📊 Send photo for STATS (caption required).")
		return
	case "🎟️ Image-post":
		ph.stateStore.SetMode(chatID, image.ModePost)
		_ = ph.bot.SendText(ctx, chatID, "🖼️ Send photo for POST (caption optional).")
		return
	}

	if ph.stateStore.IsProcessing(chatID) {
		_ = ph.bot.SendText(ctx, chatID, "😵‍💫 Slow down, I'm already inating' it!")
		return
	}

	mode := ph.stateStore.GetMode(chatID)
	if mode == image.ModeNone {
		_ = ph.bot.ShowMenu(ctx, chatID)
		return
	}

	if !hasPhoto(msg) {
		_ = ph.bot.SendText(ctx, chatID, "❌ Photo required.")
		return
	}

	if !ph.stateStore.TryStart(chatID) {
		_ = ph.bot.SendText(ctx, chatID, "😵‍💫 Slow down, I'm already inating' it!")
		return
	}

	defer ph.stateStore.Finish(chatID)

	_ = ph.processByMode(ctx, msg, mode)
}

func (ph *Handler) processByMode(ctx context.Context, msg *telego.Message, mode int) error {
	if mode == image.ModeStats {
		return ph.handleStatsPost(ctx, msg)
	}
	return ph.handleImagePost(ctx, msg)
}

func (ph *Handler) handleStatsPost(ctx context.Context, msg *telego.Message) error {
	chatID := msg.Chat.ID
	_ = ph.bot.SendText(ctx, chatID, "⏳ Statsinating...")

	resultPath, cleanup, err := ph.executeStatsPost(ctx, msg)
	if err != nil {
		return ph.fail(chatID, "executeStatsPost failed", "🚧 Error while statsinating.", err)
	}
	defer cleanup()

	return ph.bot.SendFileAuto(ctx, chatID, resultPath)
}

func (ph *Handler) executeStatsPost(ctx context.Context, msg *telego.Message) (string, func(), error) {
	title := strings.ToUpper(getText(msg))

	data, err := ph.togglService.GetMonthlyStats(ctx, title)
	if err != nil {
		return "", nil, fmt.Errorf("toggl failed: %w", err)
	}

	if len(data) == 0 {
		return "", nil, fmt.Errorf("no data")
	}

	fileID, err := extractFileID(msg)
	if err != nil {
		return "", nil, fmt.Errorf("no file: %w", err)
	}

	localImgPath, cleanupTemp, err := ph.fileManager.DownloadToTemp(ctx, fileID)
	if err != nil {
		return "", nil, fmt.Errorf("download failed: %w", err)
	}

	resultPath, err := ph.imageService.RenderStats(data, title, localImgPath)
	if err != nil {
		cleanupTemp()
		return "", nil, fmt.Errorf("render failed: %w", err)
	}

	cleanup := func() {
		cleanupTemp()
		_ = os.Remove(resultPath)
	}
	return resultPath, cleanup, nil
}

func (ph *Handler) handleImagePost(ctx context.Context, msg *telego.Message) error {
	chatID := msg.Chat.ID
	_ = ph.bot.SendText(ctx, chatID, "⏳ Postinating...")

	resultPath, cleanup, err := ph.executeImagePost(ctx, msg)
	if err != nil {
		return ph.fail(chatID, "executeImagePost failed", "🚧 Error while postinating.", err)
	}
	defer cleanup()

	return ph.bot.SendFileAuto(ctx, chatID, resultPath)
}

func (ph *Handler) executeImagePost(ctx context.Context, msg *telego.Message) (string, func(), error) {
	fileID, err := extractFileID(msg)
	if err != nil {
		return "", nil, err
	}

	localPath, cleanupTemp, err := ph.fileManager.DownloadToTemp(ctx, fileID)
	if err != nil {
		return "", nil, fmt.Errorf("download failed: %w", err)
	}

	text := getText(msg)
	resultPath, err := ph.imageService.RenderPost(localPath, text)
	if err != nil {
		cleanupTemp()
		return "", nil, fmt.Errorf("render error: %w", err)
	}

	cleanup := func() {
		cleanupTemp()
		_ = os.Remove(resultPath)
	}
	return resultPath, cleanup, nil
}

func (ph *Handler) fail(chatID int64, logMsg, userMsg string, err error) error {
	ph.logger.Printf("%s: %v", logMsg, err)
	_ = ph.bot.SendText(context.Background(), chatID, userMsg)
	return err
}

func extractFileID(msg *telego.Message) (string, error) {
	if len(msg.Photo) > 0 {
		return msg.Photo[len(msg.Photo)-1].FileID, nil
	}
	if msg.Document != nil {
		return msg.Document.FileID, nil
	}
	return "", fmt.Errorf("no file")
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
