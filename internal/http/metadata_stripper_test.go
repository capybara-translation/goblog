package http

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/chai2010/webp"
)

// createTestJPEG creates a minimal JPEG image for testing
func createTestJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

// createTestPNG creates a minimal PNG image for testing
func createTestPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// createTestGIF creates a minimal GIF image for testing
func createTestGIF() []byte {
	img := image.NewPaletted(image.Rect(0, 0, 10, 10), color.Palette{
		color.RGBA{R: 0, G: 0, B: 255, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})

	var buf bytes.Buffer
	gif.Encode(&buf, img, nil)
	return buf.Bytes()
}

// createTestWebP creates a minimal WebP image for testing
func createTestWebP() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	webp.Encode(&buf, img, &webp.Options{Lossless: true})
	return buf.Bytes()
}

func TestStripMetadata_JPEG(t *testing.T) {
	data := createTestJPEG()

	stripped, err := StripMetadata(data, "image/jpeg")
	if err != nil {
		t.Fatalf("StripMetadata failed: %v", err)
	}

	// Verify it's still a valid JPEG
	if !bytes.HasPrefix(stripped, []byte{0xFF, 0xD8, 0xFF}) {
		t.Error("Output is not a valid JPEG")
	}

	// Verify image can be decoded
	_, err = jpeg.Decode(bytes.NewReader(stripped))
	if err != nil {
		t.Errorf("Stripped JPEG cannot be decoded: %v", err)
	}
}

func TestStripMetadata_PNG(t *testing.T) {
	data := createTestPNG()

	stripped, err := StripMetadata(data, "image/png")
	if err != nil {
		t.Fatalf("StripMetadata failed: %v", err)
	}

	// Verify it's still a valid PNG (PNG signature)
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(stripped, pngSignature) {
		t.Error("Output is not a valid PNG")
	}

	// Verify image can be decoded
	_, err = png.Decode(bytes.NewReader(stripped))
	if err != nil {
		t.Errorf("Stripped PNG cannot be decoded: %v", err)
	}
}

func TestStripMetadata_GIF(t *testing.T) {
	data := createTestGIF()

	stripped, err := StripMetadata(data, "image/gif")
	if err != nil {
		t.Fatalf("StripMetadata failed: %v", err)
	}

	// Verify it's still a valid GIF
	if !bytes.HasPrefix(stripped, []byte("GIF8")) {
		t.Error("Output is not a valid GIF")
	}

	// Verify image can be decoded
	_, err = gif.Decode(bytes.NewReader(stripped))
	if err != nil {
		t.Errorf("Stripped GIF cannot be decoded: %v", err)
	}
}

func TestStripMetadata_WebP(t *testing.T) {
	data := createTestWebP()

	stripped, err := StripMetadata(data, "image/webp")
	if err != nil {
		t.Fatalf("StripMetadata failed: %v", err)
	}

	// Verify it's still a valid WebP (RIFF header)
	if !bytes.HasPrefix(stripped, []byte("RIFF")) {
		t.Error("Output is not a valid WebP (missing RIFF header)")
	}

	// Verify WEBP signature
	if len(stripped) < 12 || string(stripped[8:12]) != "WEBP" {
		t.Error("Output is not a valid WebP (missing WEBP signature)")
	}

	// Verify image can be decoded
	_, err = webp.Decode(bytes.NewReader(stripped))
	if err != nil {
		t.Errorf("Stripped WebP cannot be decoded: %v", err)
	}
}

func TestStripMetadata_UnsupportedType(t *testing.T) {
	_, err := StripMetadata([]byte("test"), "image/bmp")
	if err == nil {
		t.Error("Expected error for unsupported content type")
	}
}

func TestStripMetadata_InvalidJPEG(t *testing.T) {
	_, err := StripMetadata([]byte("not a jpeg"), "image/jpeg")
	if err == nil {
		t.Error("Expected error for invalid JPEG data")
	}
}

func TestStripMetadata_InvalidPNG(t *testing.T) {
	_, err := StripMetadata([]byte("not a png"), "image/png")
	if err == nil {
		t.Error("Expected error for invalid PNG data")
	}
}

func TestStripMetadata_InvalidGIF(t *testing.T) {
	_, err := StripMetadata([]byte("not a gif"), "image/gif")
	if err == nil {
		t.Error("Expected error for invalid GIF data")
	}
}

func TestStripMetadata_InvalidWebP(t *testing.T) {
	_, err := StripMetadata([]byte("not a webp"), "image/webp")
	if err == nil {
		t.Error("Expected error for invalid WebP data")
	}
}

func TestStripMetadata_PreservesImageDimensions(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		createFunc  func() []byte
		decodeFunc  func([]byte) (image.Image, error)
	}{
		{
			name:        "JPEG",
			contentType: "image/jpeg",
			createFunc:  createTestJPEG,
			decodeFunc: func(data []byte) (image.Image, error) {
				return jpeg.Decode(bytes.NewReader(data))
			},
		},
		{
			name:        "PNG",
			contentType: "image/png",
			createFunc:  createTestPNG,
			decodeFunc: func(data []byte) (image.Image, error) {
				return png.Decode(bytes.NewReader(data))
			},
		},
		{
			name:        "GIF",
			contentType: "image/gif",
			createFunc:  createTestGIF,
			decodeFunc: func(data []byte) (image.Image, error) {
				return gif.Decode(bytes.NewReader(data))
			},
		},
		{
			name:        "WebP",
			contentType: "image/webp",
			createFunc:  createTestWebP,
			decodeFunc: func(data []byte) (image.Image, error) {
				return webp.Decode(bytes.NewReader(data))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.createFunc()
			stripped, err := StripMetadata(original, tt.contentType)
			if err != nil {
				t.Fatalf("StripMetadata failed: %v", err)
			}

			// Decode both images and compare dimensions
			origImg, err := tt.decodeFunc(original)
			if err != nil {
				t.Fatalf("Failed to decode original: %v", err)
			}

			strippedImg, err := tt.decodeFunc(stripped)
			if err != nil {
				t.Fatalf("Failed to decode stripped: %v", err)
			}

			origBounds := origImg.Bounds()
			strippedBounds := strippedImg.Bounds()

			if origBounds.Dx() != strippedBounds.Dx() || origBounds.Dy() != strippedBounds.Dy() {
				t.Errorf("Dimensions changed: original %dx%d, stripped %dx%d",
					origBounds.Dx(), origBounds.Dy(),
					strippedBounds.Dx(), strippedBounds.Dy())
			}
		})
	}
}
