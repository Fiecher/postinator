package services

import (
	"fmt"
	img "image"
	"os"
	"path/filepath"
	"postinator/internal/files"
	"postinator/internal/image"
	"postinator/internal/toggl"
)

type ImageService struct {
	tempDir     string
	assetLoader *files.AssetLoader
	fileManager files.FileManager
}

func NewImageService(
	assetLoader *files.AssetLoader,
	fileManager files.FileManager,
	tempDir string,
) *ImageService {

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create temp directory: %v", err))
	}

	return &ImageService{
		tempDir:     tempDir,
		assetLoader: assetLoader,
		fileManager: fileManager,
	}
}

func (s *ImageService) RenderPost(inputPath, text string) (string, error) {
	assets, err := s.assetLoader.Load()
	if err != nil {
		return "", fmt.Errorf("asset load error: %w", err)
	}

	userImg, err := s.fileManager.LoadImage(inputPath)
	if err != nil {
		return "", fmt.Errorf("load user image: %w", err)
	}

	outImg, err := image.RenderPostImage(assets, userImg, text)
	if err != nil {
		return "", fmt.Errorf("render post: %w", err)
	}

	out := filepath.Join(
		s.tempDir,
		"output_"+filepath.Base(inputPath)+".jpg",
	)

	if err := image.SaveImageJPEG(out, outImg); err != nil {
		return "", fmt.Errorf("save output: %w", err)
	}

	return out, nil
}

func (s *ImageService) RenderStats(items []toggl.StatItem, title string, userImagePath string) (string, error) {
	assets, err := s.assetLoader.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load assets: %w", err)
	}

	var userImg img.Image
	if userImagePath != "" {
		uImg, err := s.fileManager.LoadImage(userImagePath)
		if err != nil {
			return "", fmt.Errorf("failed to load user image: %w", err)
		}
		userImg = uImg
	}

	outImg, err := image.RenderStatsImage(assets, items, title, userImg)
	if err != nil {
		return "", fmt.Errorf("render stats: %w", err)
	}

	outPath := filepath.Join(s.tempDir, fmt.Sprintf("stats_%d.jpg", os.Getpid()))
	if err := image.SaveImageJPEG(outPath, outImg); err != nil {
		return "", fmt.Errorf("save stats output: %w", err)
	}
	return outPath, nil
}
