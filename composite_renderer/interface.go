package composite_renderer

import "bytes"

type Renderer interface {
	TileImages(imageBufs []*bytes.Buffer) (*bytes.Buffer, error)
}
