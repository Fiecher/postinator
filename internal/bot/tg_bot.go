package bot

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mymmrac/telego"
)

type TelegramBot struct {
	client *telego.Bot
	logger *log.Logger
}

func NewTelegramBot(token string, logger *log.Logger) (*TelegramBot, error) {
	if logger == nil {
		logger = log.Default()
	}

	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telego bot: %w", err)
	}

	return &TelegramBot{
		client: bot,
		logger: logger,
	}, nil
}

type UpdateHandlerFunc func(context.Context, telego.Update)

func (tb *TelegramBot) Start(ctx context.Context, handler UpdateHandlerFunc) error {
	updates, err := tb.client.UpdatesViaLongPulling(&telego.GetUpdatesParams{Timeout: 30})
	if err != nil {
		return fmt.Errorf("failed to start long polling: %w", err)
	}

	tb.logger.Println("Bot started receiving updates...")

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				tb.logger.Println("Updates channel closed. Bot stopped.")
				return nil
			}
			go handler(ctx, update)

		case <-ctx.Done():
			tb.client.StopLongPulling()
			tb.logger.Println("Bot stopped by context cancellation.")
			return ctx.Err()
		}
	}
}

func (tb *TelegramBot) sendFileFromPath(
	ctx context.Context, chatID int64, filePath string,
	sender func(context.Context, *telego.ChatID, telego.InputFile) (*telego.Message, error)) error {

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found %s: %w", filePath, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			tb.logger.Printf("ERROR closing file %s: %v", filePath, closeErr)
		}
	}()

	_, err = sender(
		ctx,
		&telego.ChatID{ID: chatID},
		telego.InputFile{File: file},
	)

	if err != nil {
		return fmt.Errorf("failed to send file to chat %d: %w", chatID, err)
	}
	return nil
}

func (tb *TelegramBot) SendPhoto(ctx context.Context, chatID int64, filePath string) error {
	return tb.sendFileFromPath(ctx, chatID, filePath,
		func(c context.Context, id *telego.ChatID, f telego.InputFile) (*telego.Message, error) {
			return tb.client.SendPhoto(
				&telego.SendPhotoParams{
					ChatID: *id,
					Photo:  f,
				},
			)
		},
	)
}

func (tb *TelegramBot) SendDocument(ctx context.Context, chatID int64, filePath string) error {
	return tb.sendFileFromPath(ctx, chatID, filePath,
		func(c context.Context, id *telego.ChatID, f telego.InputFile) (*telego.Message, error) {
			return tb.client.SendDocument(
				&telego.SendDocumentParams{
					ChatID:   *id,
					Document: f,
				},
			)
		},
	)
}

func (tb *TelegramBot) SendText(chatID int64, text string) error {
	_, err := tb.client.SendMessage(&telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to chat %d: %w", chatID, err)
	}
	return nil
}

func (tb *TelegramBot) GetFile(fileID string) (*telego.File, error) {
	file, err := tb.client.GetFile(&telego.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for ID %s: %w", fileID, err)
	}
	return file, nil
}
