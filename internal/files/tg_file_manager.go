package files

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"postinator/internal/bot"
)

type telegramFileManager struct {
	client  bot.Bot
	tempDir string
	token   string
}

func NewTelegramFileManager(client bot.Bot, tempDir, token string) (FileManager, error) {
	if tempDir == "" {
		tempDir = "temp"
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	return &telegramFileManager{
		client:  client,
		tempDir: tempDir,
		token:   token,
	}, nil
}

func (fm *telegramFileManager) DownloadToTemp(ctx context.Context, fileID string) (string, func(), error) {
	tf, err := fm.client.GetFile(ctx, fileID)
	if err != nil {
		return "", nil, fmt.Errorf("GetFile error: %w", err)
	}
	if tf == nil || tf.FilePath == "" {
		return "", nil, fmt.Errorf("invalid file info from telegram for id %s", fileID)
	}

	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", fm.token, tf.FilePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("download request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			return "", nil, closeErr
		}
		return "", nil, fmt.Errorf("download failed: status %s, body: %s", resp.Status, string(body))
	}

	localName := filepath.Join(fm.tempDir, filepath.Base(tf.FilePath))
	out, err := os.Create(localName)
	if err != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return "", nil, closeErr
		}
		return "", nil, fmt.Errorf("failed to create local file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	if closeErr := resp.Body.Close(); closeErr != nil {
		return "", nil, closeErr
	}
	if closeErr := out.Close(); closeErr != nil {
		_ = closeErr
	}

	if err != nil {
		_ = os.Remove(localName)
		return "", nil, fmt.Errorf("failed to save downloaded file: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(localName)
	}

	return localName, cleanup, nil
}
