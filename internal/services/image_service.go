package services

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
)

type ImageService struct {
	assetsDir  string
	tempDir    string
	maxSize    int64
	background image.Image
	fontPath   string
	overlay    image.Image
}

func NewImageService(assetsDir, tempDir string, maxSize int64) *ImageService {
	service := &ImageService{
		assetsDir: assetsDir,
		tempDir:   tempDir,
		maxSize:   maxSize,
	}

	service.loadAssets()

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create temp directory: %v", err))
	}

	return service
}

func (is *ImageService) loadAssets() {
	bgPath := filepath.Join(is.assetsDir, "BG.png")
	fontPath := filepath.Join(is.assetsDir, "Buran USSR.ttf")
	overlayPath := filepath.Join(is.assetsDir, "Overlay.png")

	background, err := is.openImage(bgPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load background: %v", err))
	}
	is.background = background

	is.fontPath = fontPath

	overlay, err := is.openImage(overlayPath)
	if err != nil {
		is.overlay = nil
	} else {
		is.overlay = overlay
	}
}

func (is *ImageService) Render(inputPath, text string) (string, error) {
	background := is.background
	dc := gg.NewContextForImage(background)

	if err := is.drawText(dc, text); err != nil {
		return "", err
	}

	destImg, err := is.drawImage(dc, inputPath)
	if err != nil {
		return "", err
	}

	if is.overlay != nil {
		destImg = is.drawOverlay(destImg, is.overlay)
	}

	outputPath := filepath.Join(is.tempDir, "output_"+filepath.Base(inputPath)+".png")
	if err := is.saveImage(outputPath, destImg); err != nil {
		return "", err
	}

	return outputPath, nil
}

func (is *ImageService) drawText(context *gg.Context, text string) error {
	fontSize := float64(max(context.Width(), context.Height())) / 1000.0 * 85.0

	if err := context.LoadFontFace(is.fontPath, fontSize); err != nil {
		return fmt.Errorf("failed to load font: %w", err)
	}

	context.SetColor(color.RGBA{R: 33, G: 35, B: 50, A: 255})
	context.DrawStringAnchored(text, float64(context.Width())/2, float64(context.Height())*0.86, 0.5, 0.5)
	return nil
}

func (is *ImageService) drawImage(context *gg.Context, path string) (image.Image, error) {
	img, err := is.openImage(path)
	if err != nil {
		return nil, err
	}

	img = is.cropToSquare(img)
	targetSize := context.Width() * 6 / 10
	img = resize.Resize(uint(targetSize), uint(targetSize), img, resize.Lanczos3)

	centerX := (context.Width() - img.Bounds().Dx()) / 2
	centerY := (context.Height() - img.Bounds().Dy()) / 2

	destImg := image.NewRGBA(context.Image().Bounds())
	draw.Draw(destImg, destImg.Bounds(), context.Image(), image.Point{}, draw.Over)
	draw.Draw(destImg, img.Bounds().Add(image.Pt(centerX, centerY)), img, image.Point{}, draw.Over)

	return destImg, nil
}

func (is *ImageService) drawOverlay(baseImg image.Image, overlay image.Image) image.Image {
	baseRGBA := image.NewRGBA(baseImg.Bounds())
	draw.Draw(baseRGBA, baseRGBA.Bounds(), baseImg, image.Point{}, draw.Over)

	overlayRGBA := image.NewRGBA(overlay.Bounds())
	for y := 0; y < overlay.Bounds().Dy(); y++ {
		for x := 0; x < overlay.Bounds().Dx(); x++ {
			originalColor := overlay.At(x, y)
			r, g, b, a := originalColor.RGBA()
			alpha := uint16(float64(a) * 0.6)

			overlayRGBA.Set(x, y, color.NRGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(alpha >> 8),
			})
		}
	}

	centerX := (baseRGBA.Bounds().Dx() - overlay.Bounds().Dx()) / 2
	centerY := (baseRGBA.Bounds().Dy() - overlay.Bounds().Dy()) / 2
	draw.Draw(baseRGBA, overlay.Bounds().Add(image.Pt(centerX, centerY)), overlayRGBA, image.Point{}, draw.Over)

	return baseRGBA
}

func (is *ImageService) openImage(path string) (image.Image, error) {
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

func (is *ImageService) saveImage(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func (is *ImageService) cropToSquare(img image.Image) image.Image {
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if width == height {
		return img
	}

	cropRect := image.Rect(0, 0, width, height)
	if width > height {
		cropRect = image.Rect((width-height)/2, 0, (width+height)/2, height)
	} else {
		cropRect = image.Rect(0, (height-width)/2, width, (height+width)/2)
	}

	rgbaImg := image.NewRGBA(img.Bounds())
	draw.Draw(rgbaImg, rgbaImg.Bounds(), img, image.Point{}, draw.Over)
	return rgbaImg.SubImage(cropRect).(*image.RGBA)
}

func (is *ImageService) DownloadFile(filePath string) (io.ReadCloser, error) {
	resp, err := http.Get("https://api.telegram.org/file/bot" + os.Getenv("TOKEN") + "/" + filePath)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}

	return resp.Body, nil
}
