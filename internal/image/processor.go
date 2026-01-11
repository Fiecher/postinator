package image

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
)

type Processor struct{}

func (p *Processor) CropToSquare(img image.Image) image.Image {
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

func (p *Processor) Resize(img image.Image, size int) image.Image {
	return resize.Resize(uint(size), uint(size), img, resize.Lanczos3)
}

func (p *Processor) DrawCentered(bg image.Image, img image.Image) image.Image {
	bgBounds := bg.Bounds()
	imgBounds := img.Bounds()

	centerX := (bgBounds.Dx() - imgBounds.Dx()) / 2
	centerY := (bgBounds.Dy() - imgBounds.Dy()) / 2

	result := image.NewRGBA(bgBounds)
	draw.Draw(result, bgBounds, bg, image.Point{}, draw.Src)
	draw.Draw(result, imgBounds.Add(image.Pt(centerX, centerY)), img, image.Point{}, draw.Over)

	return result
}

func (p *Processor) DrawTextCentered(dc *gg.Context, text, fontPath string) error {
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

func (p *Processor) OverlayCentered(base image.Image, overlay image.Image, alpha float64) image.Image {
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
