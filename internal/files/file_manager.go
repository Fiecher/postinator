package files

import (
	"context"
)

type FileManager interface {
	DownloadToTemp(ctx context.Context, fileID string) (localPath string, cleanup func(), err error)
}
