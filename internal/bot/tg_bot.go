package bot

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mymmrac/telego"
)

type TelegramBot struct {
	client      *telego.Bot
	logger      *log.Logger
	maxFileSize int64
}

func NewTelegramBot(token string, logger *log.Logger, maxFileSize int64) (Bot, error) {
	if logger == nil {
		logger = log.Default()
	}

	b, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telego bot: %w", err)
	}

	return &TelegramBot{
		client:      b,
		logger:      logger,
		maxFileSize: maxFileSize,
	}, nil
}

func (tb *TelegramBot) Start(ctx context.Context, handler func(context.Context, telego.Update)) error {
	updates, err := tb.client.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{Timeout: 30})
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
			if _, err := tb.client.StopPoll(ctx, &telego.StopPollParams{}); err != nil {
				return err
			}
			tb.logger.Println("Bot stopped by context cancellation.")
			return ctx.Err()
		}
	}
}

func (tb *TelegramBot) Stop(ctx context.Context) error {
	if err := tb.client.Close(ctx); err != nil {
		return err
	}
	return nil
}

func (tb *TelegramBot) sendFileFromPath(
	ctx context.Context,
	chatID int64,
	filePath string,
	sender func(context.Context, *telego.ChatID, telego.InputFile) (*telego.Message, error),
) error {

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found %s: %w", filePath, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func(file *os.File) {
		if closeErr := file.Close(); closeErr != nil {
			tb.logger.Printf("Failed to close file %s: %w", filePath, closeErr)
		}
	}(file)

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
			return tb.client.SendPhoto(ctx, &telego.SendPhotoParams{
				ChatID: *id,
				Photo:  f,
			})
		},
	)
}

func (tb *TelegramBot) SendDocument(ctx context.Context, chatID int64, filePath string) error {
	return tb.sendFileFromPath(ctx, chatID, filePath,
		func(c context.Context, id *telego.ChatID, f telego.InputFile) (*telego.Message, error) {
			return tb.client.SendDocument(ctx, &telego.SendDocumentParams{
				ChatID:   *id,
				Document: f,
			})
		},
	)
}

func (tb *TelegramBot) SendText(ctx context.Context, chatID int64, text string) error {
	_, err := tb.client.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to chat %d: %w", chatID, err)
	}
	return nil
}

func (tb *TelegramBot) SendChatAction(ctx context.Context, chatID int64, action string) error {
	err := tb.client.SendChatAction(ctx, &telego.SendChatActionParams{
		ChatID: telego.ChatID{ID: chatID},
		Action: action,
	})
	if err != nil {
		return fmt.Errorf("failed to send chat action: %w", err)
	}
	return nil
}

func (tb *TelegramBot) GetFile(ctx context.Context, fileID string) (*File, error) {
	f, err := tb.client.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for ID %s: %w", fileID, err)
	}

	return &File{
		FileID:   f.FileID,
		FilePath: f.FilePath,
	}, nil
}

func (tb *TelegramBot) FileDownloadURL(filePath string) string {
	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", tb.client.Token, filePath)
}

func (tb *TelegramBot) SendFileAuto(ctx context.Context, chatID int64, filePath string) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	if stat.Size() <= tb.maxFileSize {
		return tb.SendPhoto(ctx, chatID, filePath)
	}
	return tb.SendDocument(ctx, chatID, filePath)
}
