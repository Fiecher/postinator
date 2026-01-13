package image

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"postinator/internal/files"
	"postinator/internal/toggl"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
	"golang.org/x/image/font"
)

func RenderPostImage(assets *files.Assets, userImg image.Image, text string) (image.Image, error) {
	if assets == nil {
		return nil, fmt.Errorf("assets is nil")
	}
	if userImg == nil {
		return nil, fmt.Errorf("user image is nil")
	}

	dc := gg.NewContextForImage(assets.Background)

	if err := drawTextCentered(dc, text, assets.FontPath); err != nil {
		return nil, fmt.Errorf("text render: %w", err)
	}

	u := cropToSquare(userImg)
	target := dc.Width() * 6 / 10
	u = resizeImage(u, target)

	composed := drawImageCentered(dc.Image(), u)

	if assets.Overlay != nil {
		composed = overlayCentered(composed, assets.Overlay, 0.6)
	}

	return composed, nil
}

func RenderStatsImage(assets *files.Assets, items []toggl.StatItem, title string, userImg image.Image) (image.Image, error) {
	if assets == nil {
		return nil, fmt.Errorf("assets is nil")
	}

	dc := gg.NewContextForImage(assets.BackgroundStats)
	W, H := float64(dc.Width()), float64(dc.Height())

	timeSize := H * 0.145
	labelSize := H * 0.05
	totalSize := H * 0.075

	timeFace, _ := gg.LoadFontFace(assets.FontPath, timeSize)
	labelFace, _ := gg.LoadFontFace(assets.FontPath, labelSize)
	totalFace, _ := gg.LoadFontFace(assets.FontPath, totalSize)

	if userImg != nil {
		drawUserStatsImage(dc, assets, userImg, W, H)
	}

	leftColX := 410.0
	rightColX := 900.0
	startY := 260.0
	rowStep := 235.0
	labelSpacing := 110.0
	maxTextWidth := timeSize * 1.8

	var totalSeconds int
	displayedItems := make([]toggl.StatItem, 0, 6)
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

		drawTimeWings(dc, x, y, timeSize, item.Color)
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

	if totalSeconds > 0 && userImg != nil {
		chartHeight := H * 0.008
		chartWidth := userImgSize
		chartX := userImgCenterX - (userImgSize / 2.0)
		chartY := userImgCenterY + (userImgSize / 2.0) + 2

		drawActivityChart(dc, displayedItems, chartX, chartY, chartWidth, chartHeight, totalSeconds)
	}

	drawFooter(dc, W, H, labelFace, totalFace, totalSeconds, title)

	return dc.Image(), nil
}

func SaveImageJPEG(path string, image image.Image) error {
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

func cropToSquare(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	if w == h {
		return img
	}

	var crop image.Rectangle
	if w > h {
		offset := (w - h) / 2
		crop = image.Rect(offset, 0, offset+h, h)
	} else {
		offset := (h - w) / 2
		crop = image.Rect(0, offset, w, offset+w)
	}

	rgba := image.NewRGBA(crop)
	draw.Draw(rgba, rgba.Bounds(), img, crop.Min, draw.Src)
	return rgba
}

func resizeImage(img image.Image, size int) image.Image {
	return resize.Resize(uint(size), uint(size), img, resize.Lanczos3)
}

func drawImageCentered(bg image.Image, img image.Image) image.Image {
	bgBounds := bg.Bounds()
	imgBounds := img.Bounds()

	centerX := (bgBounds.Dx() - imgBounds.Dx()) / 2
	centerY := (bgBounds.Dy() - imgBounds.Dy()) / 2

	result := image.NewRGBA(bgBounds)
	draw.Draw(result, bgBounds, bg, image.Point{}, draw.Src)
	draw.Draw(result, imgBounds.Add(image.Pt(centerX, centerY)), img, image.Point{}, draw.Over)

	return result
}

func drawTextCentered(dc *gg.Context, text, fontPath string) error {
	fontSize := float64(max(dc.Width(), dc.Height())) / 1000.0 * 85

	if err := dc.LoadFontFace(fontPath, fontSize); err != nil {
		return err
	}

	dc.SetRGB(0.13, 0.14, 0.2)
	dc.DrawStringAnchored(text,
		float64(dc.Width())/2,
		float64(dc.Height())*0.86,
		0.5, 0.5,
	)
	return nil
}

func overlayCentered(base image.Image, overlay image.Image, alpha float64) image.Image {
	baseRGBA := image.NewRGBA(base.Bounds())
	draw.Draw(baseRGBA, baseRGBA.Bounds(), base, image.Point{}, draw.Src)

	overlayRGBA := image.NewRGBA(overlay.Bounds())
	for y := 0; y < overlay.Bounds().Dy(); y++ {
		for x := 0; x < overlay.Bounds().Dx(); x++ {
			r, g, b, a := overlay.At(x, y).RGBA()
			a16 := uint16(float64(a) * alpha)

			overlayRGBA.Set(x, y, color.NRGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a16 >> 8),
			})
		}
	}

	centerX := (baseRGBA.Bounds().Dx() - overlay.Bounds().Dx()) / 2
	centerY := (baseRGBA.Bounds().Dy() - overlay.Bounds().Dy()) / 2

	draw.Draw(
		baseRGBA,
		overlay.Bounds().Add(image.Pt(centerX, centerY)),
		overlayRGBA,
		image.Point{},
		draw.Over,
	)

	return baseRGBA
}

func drawUserStatsImage(dc *gg.Context, assets *files.Assets, img image.Image, W, H float64) {

	centerX, centerY := W*0.75, H*0.43
	targetSize := int(H * 0.45)

	uImg := cropToSquare(img)
	uImg = resizeImage(uImg, targetSize)
	dc.DrawImageAnchored(uImg, int(centerX), int(centerY), 0.5, 0.5)

	if assets.Overlay != nil {
		overlaySize := int(float64(targetSize) * 1.04)
		overlayResized := resizeImage(assets.Overlay, overlaySize)

		dc.Push()
		dc.SetRGBA(1, 1, 1, 0.6)
		dc.DrawImageAnchored(overlayResized, int(centerX), int(centerY), 0.5, 0.5)
		dc.Pop()
	}
}

func drawActivityChart(dc *gg.Context, items []toggl.StatItem, x, y, w, h float64, totalSec int) {
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

func drawTimeWings(dc *gg.Context, x, y, fontSize float64, c color.Color) {
	dc.SetColor(c)

	scale := (fontSize / 125.5) * 0.77
	margin := fontSize - 10

	drawPointWing(dc, x-margin, y, scale, []struct{ x, y float64 }{
		{199.4, 338.1}, {236.9, 338.1}, {251.85, 209.59}, {215.42, 209.59}, {235.54, 275.79},
	}, 251.85)

	drawPointWing(dc, x+margin, y, scale, []struct{ x, y float64 }{
		{513.73, 338.16}, {551.23, 338.16}, {531.12, 275.85}, {567.24, 209.65}, {528.57, 209.65},
	}, 513.73)
}

func drawPointWing(dc *gg.Context, anchorX, anchorY, scale float64, points []struct{ x, y float64 }, refX float64) {
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

func drawFooter(dc *gg.Context, W, H float64, labelFace, totalFace font.Face, totalSec int, title string) {
	footerX := W * 0.75
	footerY := H * 0.70

	dc.SetFontFace(labelFace)
	dc.SetRGB255(33, 35, 50)
	dc.DrawStringAnchored(title, footerX, footerY, 0.5, 0.5)

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
