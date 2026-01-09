package services

import (
	"fmt"
	img "image"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"

	"postinator/internal/files"
	"postinator/internal/image"
)

type ImageService struct {
	tempDir      string
	assetLoader  *files.AssetLoader
	processor    *image.Processor
	textRenderer *image.TextRenderer
	fileManager  files.FileManager
}

func NewImageService(
	assetLoader *files.AssetLoader,
	textRenderer *image.TextRenderer,
	processor *image.Processor,
	fileManager files.FileManager,
	tempDir string,
) *ImageService {

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create temp directory: %v", err))
	}

	return &ImageService{
		tempDir:      tempDir,
		assetLoader:  assetLoader,
		textRenderer: textRenderer,
		processor:    processor,
		fileManager:  fileManager,
	}
}

func (s *ImageService) Render(inputPath, text string) (string, error) {
	assets, err := s.assetLoader.Load()
	if err != nil {
		return "", fmt.Errorf("asset load error: %w", err)
	}

	userImg, err := s.fileManager.LoadImage(inputPath)
	if err != nil {
		return "", fmt.Errorf("load user image: %w", err)
	}

	dc := gg.NewContextForImage(assets.Background)

	s.textRenderer.FontPath = assets.FontPath
	if err := s.textRenderer.DrawCentered(dc, text); err != nil {
		return "", fmt.Errorf("text render: %w", err)
	}

	userImg = s.processor.CropToSquare(userImg)

	target := dc.Width() * 6 / 10
	userImg = s.processor.Resize(userImg, target)

	composed := s.processor.DrawCentered(dc.Image(), userImg)

	if assets.Overlay != nil {
		composed = s.processor.OverlayCentered(composed, assets.Overlay, 0.6)
	}

	out := filepath.Join(
		s.tempDir,
		"output_"+filepath.Base(inputPath)+".jpg",
	)

	if err := saveImageJPEG(out, composed); err != nil {
		return "", fmt.Errorf("save output: %w", err)
	}

	return out, nil
}

func saveImageJPEG(path string, image img.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	options := &jpeg.Options{
		Quality: 100,
	}

	return jpeg.Encode(f, image, options)
}
