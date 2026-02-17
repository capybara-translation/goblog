package http

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure/v2"
	pngstructure "github.com/dsoprea/go-png-image-structure/v2"
)

// StripMetadata removes EXIF and other metadata from image data.
// Returns the stripped image data or an error.
func StripMetadata(data []byte, contentType string) ([]byte, error) {
	switch contentType {
	case "image/jpeg":
		return stripJPEGMetadata(data)
	case "image/png":
		return stripPNGMetadata(data)
	case "image/gif":
		return stripGIFMetadata(data)
	case "image/webp":
		return stripWebPMetadata(data)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}

// getJPEGOrientation reads the EXIF Orientation tag from a parsed JPEG segment list.
// Returns the orientation value (1-8). If no EXIF data or no Orientation tag is found,
// returns 1 (normal orientation).
func getJPEGOrientation(sl *jpegstructure.SegmentList) int {
	_, _, exifTags, err := sl.DumpExif()
	if err != nil {
		return 1
	}

	for _, tag := range exifTags {
		if tag.TagId == 0x0112 { // Orientation
			if shorts, ok := tag.Value.([]uint16); ok && len(shorts) > 0 {
				orientation := int(shorts[0])
				if orientation >= 1 && orientation <= 8 {
					return orientation
				}
			}
		}
	}
	return 1
}

// applyOrientation decodes a JPEG, applies the correct pixel transformation
// based on the EXIF orientation value, and re-encodes at high quality.
func applyOrientation(data []byte, orientation int) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG for orientation correction: %w", err)
	}

	var corrected image.Image
	switch orientation {
	case 2:
		corrected = imaging.FlipH(img)
	case 3:
		corrected = imaging.Rotate180(img)
	case 4:
		corrected = imaging.FlipV(img)
	case 5:
		corrected = imaging.Transpose(img)
	case 6:
		corrected = imaging.Rotate270(img)
	case 7:
		corrected = imaging.Transverse(img)
	case 8:
		corrected = imaging.Rotate90(img)
	default:
		return data, nil
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, corrected, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("failed to re-encode JPEG after orientation correction: %w", err)
	}

	return buf.Bytes(), nil
}

// stripJPEGMetadata corrects EXIF orientation (if needed) and removes EXIF data.
// iPhone photos store orientation in EXIF metadata; when stripping EXIF, the
// orientation must first be baked into the pixel data to prevent rotated display.
func stripJPEGMetadata(data []byte) ([]byte, error) {
	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JPEG: %w", err)
	}

	sl := intfc.(*jpegstructure.SegmentList)

	// Read orientation BEFORE dropping EXIF
	orientation := getJPEGOrientation(sl)

	// If orientation needs correction, decode + transform + re-encode
	if orientation != 1 {
		data, err = applyOrientation(data, orientation)
		if err != nil {
			return nil, fmt.Errorf("failed to apply orientation correction: %w", err)
		}
		// Re-parse the corrected JPEG
		intfc, err = jmp.ParseBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse corrected JPEG: %w", err)
		}
		sl = intfc.(*jpegstructure.SegmentList)
	}

	// Drop EXIF data (APP1 segment)
	_, err = sl.DropExif()
	if err != nil {
		// ErrNoExif is not a real error - just means no EXIF to remove
		if err.Error() != "no exif data" {
			return nil, fmt.Errorf("failed to drop EXIF: %w", err)
		}
	}

	// Write back without re-encoding
	var buf bytes.Buffer
	if err := sl.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write JPEG: %w", err)
	}

	return buf.Bytes(), nil
}

// stripPNGMetadata removes metadata chunks from PNG without re-encoding.
func stripPNGMetadata(data []byte) ([]byte, error) {
	pmp := pngstructure.NewPngMediaParser()
	intfc, err := pmp.ParseBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PNG: %w", err)
	}

	cs := intfc.(*pngstructure.ChunkSlice)
	chunks := cs.Chunks()

	// Metadata chunk types to remove
	metadataChunks := map[string]bool{
		"eXIf": true, // EXIF data
		"tEXt": true, // Textual data
		"iTXt": true, // International textual data
		"zTXt": true, // Compressed textual data
	}

	// Filter out metadata chunks
	var filteredChunks []*pngstructure.Chunk
	for _, chunk := range chunks {
		if !metadataChunks[chunk.Type] {
			filteredChunks = append(filteredChunks, chunk)
		}
	}

	// Write filtered chunks
	var buf bytes.Buffer

	// Write PNG signature
	buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})

	for _, chunk := range filteredChunks {
		if _, err := chunk.WriteTo(&buf); err != nil {
			return nil, fmt.Errorf("failed to write PNG chunk: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// stripGIFMetadata removes metadata from GIF by re-encoding.
// GIF is a lossless format, so quality is preserved.
func stripGIFMetadata(data []byte) ([]byte, error) {
	// Decode GIF (comment extensions are not preserved through decode/encode)
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode GIF: %w", err)
	}

	// Re-encode without metadata
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, fmt.Errorf("failed to encode GIF: %w", err)
	}

	return buf.Bytes(), nil
}

// stripWebPMetadata removes metadata from WebP by re-encoding.
// Uses lossless encoding to preserve quality.
func stripWebPMetadata(data []byte) ([]byte, error) {
	// Decode WebP image
	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode WebP: %w", err)
	}

	// Re-encode without metadata using lossless mode
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{
		Lossless: true, // Preserve quality
	}); err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}

	return buf.Bytes(), nil
}
