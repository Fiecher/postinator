package files

import (
	"image"
	"os"
	"path/filepath"
)

type AssetLoader struct {
	bgPath      string
	fontPath    string
	overlayPath string
}

func NewAssetLoader(assetsDir, bgFile, fontFile, overlayFile string) *AssetLoader {
	return &AssetLoader{
		bgPath:      filepath.Join(assetsDir, bgFile),
		fontPath:    filepath.Join(assetsDir, fontFile),
		overlayPath: filepath.Join(assetsDir, overlayFile),
	}
}

func openImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (l *AssetLoader) Load() (*Assets, error) {
	bg, err := openImage(l.bgPath)
	if err != nil {
		return nil, err
	}

	overlay, _ := openImage(l.overlayPath)

	return &Assets{
		Background: bg,
		Overlay:    overlay,
		FontPath:   l.fontPath,
	}, nil
}
