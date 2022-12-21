// adapted from https://github.com/parsiya/Go-Security/blob/master/png-tests/png-chunk-extraction.go

package png_info_extractor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
)

// 89 50 4E 47 0D 0A 1A 0A
var pngHeader = "\x89\x50\x4E\x47\x0D\x0A\x1A\x0A"
var iHDRlength = 13

// uInt32ToInt converts a 4 byte big-endian buffer to int.
func uInt32ToInt(buf []byte) (int, error) {
	if len(buf) == 0 || len(buf) > 4 {
		return 0, errors.New("invalid buffer")
	}

	return int(binary.BigEndian.Uint32(buf)), nil
}

// Each chunk starts with a uint32 length (big endian), then 4 byte name,
// then data and finally the CRC32 of the chunk data.
type chunk struct {
	Length int    // chunk data length
	CType  string // chunk type
	Data   []byte // chunk data
	Crc32  []byte // CRC32 of chunk data
}

// populate will read bytes from the reader and populate a chunk.
func (c *chunk) populate(r io.Reader) error {
	// Four byte buffer.
	buf := make([]byte, 4)

	// Read first four bytes == chunk length.
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}

	// Convert bytes to int.
	// c.length = int(binary.BigEndian.Uint32(buf))
	var err error

	c.Length, err = uInt32ToInt(buf)
	if err != nil {
		return errors.New("cannot convert length to int")
	}

	// Read second four bytes == chunk type.
	if _, err = io.ReadFull(r, buf); err != nil {
		return err
	}

	c.CType = string(buf)

	// Read chunk data.
	tmp := make([]byte, c.Length)

	if _, err = io.ReadFull(r, tmp); err != nil {
		return err
	}

	c.Data = tmp

	// Read CRC32 hash
	if _, err = io.ReadFull(r, buf); err != nil {
		return err
	}

	// We don't really care about checking the hash.
	c.Crc32 = buf

	return nil
}

// -----------

type png struct {
	Width             int
	Height            int
	BitDepth          int
	ColorType         int
	CompressionMethod int
	FilterMethod      int
	InterlaceMethod   int
	chunks            []*chunk // Not exported == won't appear in JSON string.
	NumberOfChunks    int
}

// Parse IHDR chunk.
// https://golang.org/src/image/png/reader.go?#L142 is your friend.
func (png *png) parseIHDR(iHDR *chunk) error {
	if iHDR.Length != iHDRlength {
		errString := fmt.Sprintf("invalid IHDR length: got %d - expected %d",
			iHDR.Length, iHDRlength)

		return errors.New(errString)
	}

	// IHDR: http://www.libpng.org/pub/png/spec/1.2/PNG-Chunks.html#C.IHDR

	// Width:              4 bytes
	// Height:             4 bytes
	// Bit depth:          1 byte
	// Color type:         1 byte
	// Compression method: 1 byte
	// Filter method:      1 byte
	// Interlace method:   1 byte

	tmp := iHDR.Data

	var err error

	png.Width, err = uInt32ToInt(tmp[0:4])
	if err != nil || png.Width <= 0 {
		errString := fmt.Sprintf("invalid width in iHDR - got %x", tmp[0:4])

		return errors.New(errString)
	}

	png.Height, err = uInt32ToInt(tmp[4:8])
	if err != nil || png.Height <= 0 {
		errString := fmt.Sprintf("invalid height in iHDR - got %x", tmp[4:8])

		return errors.New(errString)
	}

	png.BitDepth = int(tmp[8])
	png.ColorType = int(tmp[9])

	// Only compression method 0 is supported
	if int(tmp[10]) != 0 {
		errString := fmt.Sprintf("invalid compression method - expected 0 - got %x",
			tmp[10])

		return errors.New(errString)
	}
	png.CompressionMethod = int(tmp[10])

	// Only filter method 0 is supported
	if int(tmp[11]) != 0 {
		errString := fmt.Sprintf("invalid filter method - expected 0 - got %x",
			tmp[11])

		return errors.New(errString)
	}
	png.FilterMethod = int(tmp[11])

	// Only interlace methods 0 and 1 are supported
	if int(tmp[12]) != 0 {
		errString := fmt.Sprintf("invalid interlace method - expected 0 or 1 - got %x",
			tmp[12])

		return errors.New(errString)
	}

	png.InterlaceMethod = int(tmp[12])

	return nil
}

// Populate populates the PNG fields (and other fields).
func (png *png) Populate() error {
	if err := png.parseIHDR(png.chunks[0]); err != nil {
		return err
	}

	png.NumberOfChunks = len(png.chunks)

	return nil
}

// ExportChunk returns a chunk's data as []byte. Numbering starts from 0.
func (png png) ExportChunk(chunkNumber int) ([]byte, error) {
	if chunkNumber >= png.NumberOfChunks {
		errString := fmt.Sprintf("invalid chunk number. Got: %d, "+
			"file has %d chunks. Chunk numbers starts from zero.",
			chunkNumber, len(png.chunks))
		return nil, errors.New(errString)
	}
	return png.chunks[chunkNumber].Data, nil
}

type extractorImpl struct {
	png *png
}

type Config struct {
	PngData []byte
}

func New(cfg Config) (Extractor, error) {
	if cfg.PngData == nil {
		return nil, errors.New("png data is nil")
	}

	// Read first 8 bytes == PNG header.
	header := make([]byte, 8)

	imgFile := bytes.NewReader(cfg.PngData)

	// Read CRC32 hash
	if _, err := io.ReadFull(imgFile, header); err != nil {
		log.Printf("Error reading PNG header: %v", err)

		return nil, err
	}

	if string(header) != pngHeader {
		log.Printf("Wrong PNG header.\nGot %x - Expected %x\n", header, pngHeader)

		return nil, errors.New("wrong PNG header")
	}

	var pngImage png

	// Reset err
	var err error

	for err == nil {
		var c chunk

		err = (&c).populate(imgFile)

		// Drop the last empty chunk.
		if c.CType != "" {
			pngImage.chunks = append(pngImage.chunks, &c)
		}
	}

	if err = pngImage.Populate(); err != nil {
		log.Println("Failed to populate PNG fields.")

		return nil, err
	}

	return &extractorImpl{
		png: &pngImage,
	}, nil
}

type PNGInfo struct {
	Prompt string
}

func (e *extractorImpl) ExtractDiffusionInfo() (*PNGInfo, error) {
	for i, c := range e.png.chunks {
		if c.CType != "tEXt" {
			continue
		}

		chunkData, err := e.png.ExportChunk(i)
		if err != nil {
			log.Printf("Failed to export chunk %d: %v", i, err)

			return nil, err
		}

		chunkString := string(chunkData)

		log.Printf("PNG Info Chunk: %s", chunkString)

		if strings.HasPrefix(chunkString, "parameters\u0000") {
			prompt := strings.ReplaceAll(strings.Split(chunkString, "\n")[0], "parameters\u0000", "")

			return &PNGInfo{
				Prompt: prompt,
			}, nil
		}
	}

	return &PNGInfo{
		Prompt: "",
	}, nil
}
