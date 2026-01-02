package bot

import (
	"context"

	"github.com/mymmrac/telego"
)

type Bot interface {
	Start(ctx context.Context, handler func(context.Context, telego.Update)) error

	SendText(ctx context.Context, chatID int64, text string) error
	SendPhoto(ctx context.Context, chatID int64, filePath string) error
	SendDocument(ctx context.Context, chatID int64, filePath string) error
	SendChatAction(ctx context.Context, chatID int64, action string) error
	SendFileAuto(ctx context.Context, chatID int64, filePath string) error

	GetFile(ctx context.Context, fileID string) (*File, error)
	FileDownloadURL(filePath string) string
}
