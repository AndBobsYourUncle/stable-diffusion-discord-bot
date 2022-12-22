package composite_renderer

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/png"
)

type rendererImpl struct{}

type Config struct{}

func New(cfg Config) (Renderer, error) {
	return &rendererImpl{}, nil
}

func (r *rendererImpl) TileImages(imageBufs []*bytes.Buffer) (*bytes.Buffer, error) {
	if len(imageBufs) != 4 {
		return nil, errors.New("invalid number of images")
	}

	images := make([]image.Image, 4)

	for i, buf := range imageBufs {
		img, _, err := image.Decode(buf)
		if err != nil {
			return nil, err
		}

		images[i] = img
	}

	firstBounds := images[0].Bounds()

	for _, img := range images {
		if img.Bounds() != firstBounds {
			return nil, errors.New("images are not the same size")
		}
	}

	retImage := image.NewRGBA(image.Rect(0, 0, firstBounds.Max.X*2, firstBounds.Max.Y*2))

	draw.Draw(retImage, images[0].Bounds().Add(image.Pt(0, 0)), images[0], image.Point{}, draw.Over)
	draw.Draw(retImage, images[1].Bounds().Add(image.Pt(firstBounds.Max.X, 0)), images[1], image.Point{}, draw.Over)
	draw.Draw(retImage, images[2].Bounds().Add(image.Pt(0, firstBounds.Max.Y)), images[2], image.Point{}, draw.Over)
	draw.Draw(retImage, images[3].Bounds().Add(image.Pt(firstBounds.Max.X, firstBounds.Max.Y)), images[3], image.Point{}, draw.Over)

	imageBuf := new(bytes.Buffer)

	err := png.Encode(imageBuf, retImage)
	if err != nil {
		return nil, err
	}

	return imageBuf, nil
}
