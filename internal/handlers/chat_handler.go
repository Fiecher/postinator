package handlers

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"os"
	"postinator/internal/image"
	"postinator/internal/services"
	"strings"

	"postinator/internal/bot"
	"postinator/internal/files"

	"github.com/mymmrac/telego"
)

type Handler struct {
	imageService *services.ImageService
	bot          bot.Bot
	fileManager  files.FileManager
	stateStore   *image.RenderStateStore
	logger       *log.Logger
}

func NewHandler(
	imageService *services.ImageService,
	bot bot.Bot,
	fileManager files.FileManager,
	stateStore *image.RenderStateStore,
	logger *log.Logger,
) *Handler {
	return &Handler{
		imageService: imageService,
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
		_ = ph.bot.SendText(ctx, chatID, "📊 Send photo for STATS (caption optional)")
		return
	case "🎟️ Image-post":
		ph.stateStore.SetMode(chatID, image.ModePost)
		_ = ph.bot.SendText(ctx, chatID, "🖼️ Send photo for POST (caption optional)")
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
	_ = ph.bot.SendText(ctx, chatID, "✅ Inated it successfully!")
}

func (ph *Handler) processByMode(ctx context.Context, msg *telego.Message, mode int) error {
	if mode == image.ModeStats {
		return ph.executeStatsRender(ctx, msg)
	}
	return ph.handleImagePost(ctx, msg)
}

func (ph *Handler) executeStatsRender(ctx context.Context, msg *telego.Message) error {
	chatID := msg.Chat.ID
	title := strings.ToUpper(getText(msg))

	_ = ph.bot.SendText(ctx, chatID, "⏳ Statsinating...")

	data := ph.getMockData()
	fileID, _ := extractFileID(msg)

	localImgPath, cleanupTemp, err := ph.fileManager.DownloadToTemp(ctx, fileID)
	if err != nil {
		return ph.fail(chatID, "download failed", "🚧 Error downloading services", err)
	}
	defer cleanupTemp()

	resultPath, err := ph.imageService.RenderStats(data, title, localImgPath)
	if err != nil {
		return ph.fail(chatID, "render failed", "❌ Error rendering stats", err)
	}
	defer os.Remove(resultPath)

	return ph.bot.SendFileAuto(ctx, chatID, resultPath)
}

func (ph *Handler) handleImagePost(ctx context.Context, msg *telego.Message) error {
	chatID := msg.Chat.ID
	_ = ph.bot.SendText(ctx, chatID, "⏳ Postinating...")

	resultPath, cleanup, err := ph.executeImagePost(ctx, msg)
	if err != nil {
		return ph.fail(chatID, "executeImagePost failed", "🚧 Error processing services", err)
	}
	defer cleanup()

	return ph.bot.SendFileAuto(ctx, chatID, resultPath)
}

func (ph *Handler) getMockData() []image.StatItem {
	return []image.StatItem{
		{Label: "writing", Duration: "555:14", Color: color.RGBA{R: 242, G: 201, B: 76, A: 255}},
		{Label: "blender", Duration: "12:54", Color: color.RGBA{R: 242, G: 153, B: 74, A: 255}},
		{Label: "java", Duration: "15:54", Color: color.RGBA{R: 235, G: 87, B: 87, A: 255}},
		{Label: "gym", Duration: "12:53", Color: color.RGBA{R: 47, G: 128, B: 237, A: 255}},
		{Label: "reading", Duration: "13:04", Color: color.RGBA{R: 39, G: 174, B: 96, A: 255}},
		{Label: "other", Duration: "01:20", Color: color.RGBA{R: 211, G: 84, B: 0, A: 255}},
	}
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
