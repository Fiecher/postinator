package files

import (
	"context"
	"image"
)

type FileManager interface {
	DownloadToTemp(ctx context.Context, fileID string) (localPath string, cleanup func(), err error)
	LoadImage(path string) (image.Image, error)
}
