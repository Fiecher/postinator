package handlers

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"os"
	"postinator/internal/bot"
	"postinator/internal/files"
	"postinator/internal/services"
	"postinator/internal/storage"
	"strings"

	"github.com/mymmrac/telego"
)

const (
	StepNone = iota
	StepAwaitingStatsTitle
	StepAwaitingStatsPhoto
)

type PhotoHandler struct {
	imageService *services.ImageService
	bot          bot.Bot
	fileManager  files.FileManager
	stateStore   *storage.RenderStateStore
	logger       *log.Logger

	statsTitles map[int64]string
	statsSteps  map[int64]int
}

func NewPhotoHandler(
	imageService *services.ImageService,
	bot bot.Bot,
	fileManager files.FileManager,
	stateStore *storage.RenderStateStore,
	logger *log.Logger,
) *PhotoHandler {
	return &PhotoHandler{
		imageService: imageService,
		bot:          bot,
		fileManager:  fileManager,
		stateStore:   stateStore,
		logger:       logger,
		statsTitles:  make(map[int64]string),
		statsSteps:   make(map[int64]int),
	}
}

func (ph *PhotoHandler) HandleUpdate(ctx context.Context, update telego.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	chatID := msg.Chat.ID

	switch msg.Text {
	case "/start":
		ph.resetStatsState(chatID)
		_ = ph.bot.ShowMenu(ctx, chatID)
		return
	case "🎫 Monthly-post":
		ph.statsSteps[chatID] = StepAwaitingStatsTitle
		_ = ph.bot.SendText(ctx, chatID, "📊 Write title for stats:")
		return
	case "🎟️ Image-post":
		ph.resetStatsState(chatID)
		_ = ph.bot.SendText(ctx, chatID, "🔗 Please send a photo or document with caption (optional).")
		return
	}

	if step := ph.statsSteps[chatID]; step != StepNone {
		ph.handleStatsDialogue(ctx, msg, step)
		return
	}

	if !hasPhoto(msg) {
		if ph.stateStore.IsProcessing(chatID) {
			_ = ph.bot.SendText(ctx, chatID, "😵‍💫 Slow down, I'm already postinatin' it.")
		} else {
			_ = ph.bot.SendText(ctx, chatID, "🔗 Please send a photo or document with caption (optional).")
		}
		return
	}

	_ = ph.withProcessing(ctx, chatID, func() error {
		return ph.handleImagePost(ctx, msg)
	})
}

func (ph *PhotoHandler) handleStatsDialogue(ctx context.Context, msg *telego.Message, step int) {
	chatID := msg.Chat.ID

	if step == StepAwaitingStatsTitle {
		if msg.Text == "" {
			_ = ph.bot.SendText(ctx, chatID, "❌ Please, send title text.")
			return
		}
		ph.statsTitles[chatID] = strings.ToUpper(msg.Text)
		ph.statsSteps[chatID] = StepAwaitingStatsPhoto
		_ = ph.bot.SendText(ctx, chatID, fmt.Sprintf("✅ Title «%s» accepted.\n🔗 Please send a photo or document.", ph.statsTitles[chatID]))
		return
	}

	if step == StepAwaitingStatsPhoto {
		if !hasPhoto(msg) {
			_ = ph.bot.SendText(ctx, chatID, "❌ Please, send photo or document.")
			return
		}

		_ = ph.withProcessing(ctx, chatID, func() error {
			return ph.executeStatsRender(ctx, msg)
		})
	}
}

func (ph *PhotoHandler) executeStatsRender(ctx context.Context, msg *telego.Message) error {
	chatID := msg.Chat.ID
	title := ph.statsTitles[chatID]

	_ = ph.bot.SendText(ctx, chatID, "⏳ Postinating stats...")

	data := ph.getMockData()

	fileID, _ := extractFileID(msg)
	localImgPath, cleanupTemp, err := ph.fileManager.DownloadToTemp(ctx, fileID)
	if err != nil {
		return ph.fail(chatID, "download failed", "🚧 Error downloading image", err)
	}
	defer cleanupTemp()

	resultPath, err := ph.imageService.RenderStats(data, title, localImgPath)
	if err != nil {
		return ph.fail(chatID, "stats render failed", "❌ Error rendering stats", err)
	}
	defer os.Remove(resultPath)

	if err := ph.bot.SendFileAuto(ctx, chatID, resultPath); err != nil {
		return ph.fail(chatID, "send error", "🚧 Error sending stats", err)
	}

	ph.resetStatsState(chatID)
	return ph.bot.SendText(ctx, chatID, "✅ Stats postinated successfully!")
}

func (ph *PhotoHandler) handleImagePost(ctx context.Context, msg *telego.Message) error {
	chatID := msg.Chat.ID
	ph.logger.Println("Postinating image post...")

	resultPath, cleanup, err := ph.process(ctx, msg)
	if err != nil {
		return ph.fail(chatID, "processing failed", "🚧 Error processing image", err)
	}
	defer cleanup()

	if err := ph.bot.SendFileAuto(ctx, chatID, resultPath); err != nil {
		return ph.fail(chatID, "send error", "🚧 Error sending result", err)
	}

	return ph.bot.SendText(ctx, chatID, "✅ Image postinated successfully!")
}

func (ph *PhotoHandler) resetStatsState(chatID int64) {
	delete(ph.statsTitles, chatID)
	delete(ph.statsSteps, chatID)
}

func (ph *PhotoHandler) getMockData() []services.StatItem {
	return []services.StatItem{
		{Label: "writing", Duration: "555:14", Color: color.RGBA{R: 242, G: 201, B: 76, A: 255}},
		{Label: "blender", Duration: "12:54", Color: color.RGBA{R: 242, G: 153, B: 74, A: 255}},
		{Label: "java", Duration: "15:54", Color: color.RGBA{R: 235, G: 87, B: 87, A: 255}},
		{Label: "gym", Duration: "12:53", Color: color.RGBA{R: 47, G: 128, B: 237, A: 255}},
		{Label: "reading", Duration: "13:04", Color: color.RGBA{R: 39, G: 174, B: 96, A: 255}},
		{Label: "other", Duration: "01:20", Color: color.RGBA{R: 211, G: 84, B: 0, A: 255}},
	}
}

func (ph *PhotoHandler) process(ctx context.Context, msg *telego.Message) (string, func(), error) {
	fileID, err := extractFileID(msg)
	if err != nil {
		return "", nil, err
	}

	localPath, cleanupTemp, err := ph.fileManager.DownloadToTemp(ctx, fileID)
	if err != nil {
		return "", nil, fmt.Errorf("download failed: %w", err)
	}

	text := getText(msg)
	resultPath, err := ph.imageService.Render(localPath, text)
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

func (ph *PhotoHandler) withProcessing(ctx context.Context, chatID int64, fn func() error) error {
	if !ph.stateStore.TryStart(chatID) {
		_ = ph.bot.SendText(ctx, chatID, "😵‍💫 Slow down, I'm already postinatin' it.")
		return fmt.Errorf("already processing")
	}
	defer ph.stateStore.Finish(chatID)
	return fn()
}

func (ph *PhotoHandler) fail(chatID int64, logMsg, userMsg string, err error) error {
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
