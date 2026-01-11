package services

import (
	"fmt"
	img "image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"

	"postinator/internal/files"
	"postinator/internal/image"
)

type ImageService struct {
	tempDir     string
	assetLoader *files.AssetLoader
	processor   *image.Processor
	fileManager files.FileManager
}

func NewImageService(
	assetLoader *files.AssetLoader,
	processor *image.Processor,
	fileManager files.FileManager,
	tempDir string,
) *ImageService {

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create temp directory: %v", err))
	}

	return &ImageService{
		tempDir:     tempDir,
		assetLoader: assetLoader,
		processor:   processor,
		fileManager: fileManager,
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

	if err := s.processor.DrawTextCentered(dc, text, assets.FontPath); err != nil {
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

type StatItem struct {
	Label    string
	Duration string
	Color    color.RGBA
}

func (s *ImageService) RenderStats(items []StatItem, title string, userImagePath string) (string, error) {
	assets, err := s.assetLoader.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load assets: %w", err)
	}

	dc := gg.NewContextForImage(assets.BackgroundStats)
	W, H := float64(dc.Width()), float64(dc.Height())

	timeSize := H * 0.145
	labelSize := H * 0.05
	totalSize := H * 0.075

	timeFace, _ := gg.LoadFontFace(assets.FontPath, timeSize)
	labelFace, _ := gg.LoadFontFace(assets.FontPath, labelSize)
	totalFace, _ := gg.LoadFontFace(assets.FontPath, totalSize)

	if userImagePath != "" {
		s.drawUserStatsImage(dc, assets, userImagePath, W, H)
	}

	leftColX := 410.0
	rightColX := 900.0
	startY := 260.0
	rowStep := 235.0
	labelSpacing := 110.0
	maxTextWidth := timeSize * 1.8

	var totalSeconds int
	displayedItems := make([]StatItem, 0, 6)
	for i, item := range items {
		if i >= 6 {
			break
		}
		displayedItems = append(displayedItems, item)
		totalSeconds += parseDurationToSeconds(item.Duration)

		var x, y float64
		if i < 3 {
			x = leftColX
			y = startY + float64(i)*rowStep
		} else {
			x = rightColX
			y = startY + float64(i-3)*rowStep
		}

		s.drawTimeWings(dc, x, y, timeSize, item.Color)
		dc.SetFontFace(timeFace)
		dc.SetColor(item.Color)

		textW, _ := dc.MeasureString(item.Duration)
		dc.Push()
		dc.Translate(x, y)
		if textW > maxTextWidth {
			dc.Scale(maxTextWidth/textW, 1.0)
		}
		dc.DrawStringAnchored(item.Duration, 0, 0, 0.5, 0.5)
		dc.Pop()

		dc.SetFontFace(labelFace)
		dc.SetRGB255(20, 30, 40)
		dc.DrawStringAnchored(item.Label, x, y+labelSpacing, 0.5, 0.5)
	}

	userImgCenterX := W * 0.75
	userImgCenterY := H * 0.42
	userImgSize := H * 0.45

	if totalSeconds > 0 && userImagePath != "" {
		chartHeight := H * 0.008
		chartWidth := userImgSize
		chartX := userImgCenterX - (userImgSize / 2.0)
		chartY := userImgCenterY + (userImgSize / 2.0) - 7

		s.drawActivityChart(dc, displayedItems, chartX, chartY, chartWidth, chartHeight, totalSeconds)
	}

	s.drawFooter(dc, W, H, labelFace, totalFace, totalSeconds, title)

	outPath := filepath.Join(s.tempDir, fmt.Sprintf("stats_%d.jpg", os.Getpid()))
	if err := saveImageJPEG(outPath, dc.Image()); err != nil {
		return "", err
	}
	return outPath, nil
}

func (s *ImageService) drawUserStatsImage(dc *gg.Context, assets *files.Assets, path string, W, H float64) {
	uImg, err := s.fileManager.LoadImage(path)
	if err != nil {
		return
	}

	centerX, centerY := W*0.75, H*0.42
	targetSize := int(H * 0.45)

	uImg = s.processor.CropToSquare(uImg)
	uImg = s.processor.Resize(uImg, targetSize)
	dc.DrawImageAnchored(uImg, int(centerX), int(centerY), 0.5, 0.5)

	if assets.Overlay != nil {
		overlaySize := int(float64(targetSize) * 1.04)
		overlayResized := s.processor.Resize(assets.Overlay, overlaySize)

		dc.Push()
		dc.SetRGBA(1, 1, 1, 0.6)
		dc.DrawImageAnchored(overlayResized, int(centerX), int(centerY), 0.5, 0.5)
		dc.Pop()
	}
}

func (s *ImageService) drawActivityChart(dc *gg.Context, items []StatItem, x, y, w, h float64, totalSec int) {
	if totalSec <= 0 {
		return
	}

	currentX := x
	totalWidthFloat := w
	totalSecFloat := float64(totalSec)

	for _, item := range items {
		itemSec := parseDurationToSeconds(item.Duration)
		if itemSec <= 0 {
			continue
		}

		ratio := float64(itemSec) / totalSecFloat
		segmentWidth := totalWidthFloat * ratio

		dc.SetColor(item.Color)
		dc.DrawRectangle(currentX, y, segmentWidth, h)
		dc.Fill()

		currentX += segmentWidth
	}
}

func (s *ImageService) drawTimeWings(dc *gg.Context, x, y, fontSize float64, c color.Color) {
	dc.SetColor(c)

	scale := (fontSize / 125.5) * 0.77
	margin := fontSize

	s.drawPointWing(dc, x-margin, y, scale, []struct{ x, y float64 }{
		{199.4, 338.1}, {236.9, 338.1}, {251.85, 209.59}, {215.42, 209.59}, {235.54, 275.79},
	}, 251.85)

	s.drawPointWing(dc, x+margin, y, scale, []struct{ x, y float64 }{
		{513.73, 338.16}, {551.23, 338.16}, {531.12, 275.85}, {567.24, 209.65}, {528.57, 209.65},
	}, 513.73)
}

func (s *ImageService) drawPointWing(dc *gg.Context, anchorX, anchorY, scale float64, points []struct{ x, y float64 }, refX float64) {
	const refY = 255.85

	for i, p := range points {
		dx := (p.x - refX) * scale
		dy := (p.y - refY) * scale
		if i == 0 {
			dc.MoveTo(anchorX+dx, anchorY+dy)
		} else {
			dc.LineTo(anchorX+dx, anchorY+dy)
		}
	}
	dc.ClosePath()
	dc.Fill()
}

func (s *ImageService) drawSingleWing(dc *gg.Context, x, y, h, w float64, isRight bool) {
	mult := 1.0
	if isRight {
		mult = -1.0
	}

	dc.MoveTo(x, y-h*0.5)
	dc.LineTo(x-w*mult, y-h*0.5)
	dc.LineTo(x-(w*0.6)*mult, y)
	dc.LineTo(x-w*mult, y+h*0.5)
	dc.LineTo(x, y+h*0.5)
	dc.LineTo(x+(w*0.2)*mult, y)
	dc.ClosePath()
	dc.Fill()
}

func (s *ImageService) drawFooter(dc *gg.Context, W, H float64, labelFace, totalFace font.Face, totalSec int, title string) {
	footerX := W * 0.75
	footerY := H * 0.70

	dc.SetFontFace(labelFace)
	dc.SetRGB255(33, 35, 50)
	dc.DrawStringAnchored(strings.ToUpper(title), footerX, footerY, 0.5, 0.5)

	totalStr := formatSecondsToDuration(totalSec)
	dc.SetFontFace(totalFace)
	dc.SetRGB255(135, 255, 198)
	dc.DrawStringAnchored(totalStr, footerX, footerY+(H*0.065), 0.5, 0.5)
}

func parseDurationToSeconds(d string) int {
	parts := strings.Split(d, ":")
	var h, m, s int
	if len(parts) == 3 {
		h, _ = strconv.Atoi(parts[0])
		m, _ = strconv.Atoi(parts[1])
		s, _ = strconv.Atoi(parts[2])
	} else if len(parts) == 2 {
		h, _ = strconv.Atoi(parts[0])
		m, _ = strconv.Atoi(parts[1])
	}
	return h*3600 + m*60 + s
}

func formatSecondsToDuration(total int) string {
	h := total / 3600
	m := (total % 3600) / 60
	return fmt.Sprintf("%02d:%02d", h, m)
}
