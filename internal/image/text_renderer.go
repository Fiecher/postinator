package image

import "github.com/fogleman/gg"

type TextRenderer struct {
	FontPath string
}

func (tr *TextRenderer) DrawCentered(dc *gg.Context, text string) error {
	fontSize := float64(max(dc.Width(), dc.Height())) / 1000.0 * 85

	if err := dc.LoadFontFace(tr.FontPath, fontSize); err != nil {
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
