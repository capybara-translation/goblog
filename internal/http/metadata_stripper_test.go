package http

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/chai2010/webp"
	jpegstructure "github.com/dsoprea/go-jpeg-image-structure/v2"
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

// createTestJPEGWithOrientation creates a 20x10 JPEG with EXIF Orientation set.
// The image has distinct quadrant colors for visual verification:
//
//	Top-left=Red, Top-right=Green, Bottom-left=Blue, Bottom-right=White
func createTestJPEGWithOrientation(orientation uint16) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	for y := 0; y < 5; y++ {
		for x := 10; x < 20; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	for y := 5; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}
	for y := 5; y < 10; y++ {
		for x := 10; x < 20; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	var jpegBuf bytes.Buffer
	jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: 95})

	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(jpegBuf.Bytes())
	if err != nil {
		panic(fmt.Sprintf("failed to parse test JPEG: %v", err))
	}
	sl := intfc.(*jpegstructure.SegmentList)

	rootIb, err := sl.ConstructExifBuilder()
	if err != nil {
		panic(fmt.Sprintf("failed to construct EXIF builder: %v", err))
	}

	err = rootIb.SetStandardWithName("Orientation", []uint16{orientation})
	if err != nil {
		panic(fmt.Sprintf("failed to set orientation: %v", err))
	}

	err = sl.SetExif(rootIb)
	if err != nil {
		panic(fmt.Sprintf("failed to set EXIF: %v", err))
	}

	var outBuf bytes.Buffer
	err = sl.Write(&outBuf)
	if err != nil {
		panic(fmt.Sprintf("failed to write test JPEG: %v", err))
	}

	return outBuf.Bytes()
}

func TestGetJPEGOrientation(t *testing.T) {
	tests := []struct {
		name        string
		orientation uint16
		expected    int
	}{
		{"orientation 1", 1, 1},
		{"orientation 3", 3, 3},
		{"orientation 6", 6, 6},
		{"orientation 8", 8, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := createTestJPEGWithOrientation(tt.orientation)
			jmp := jpegstructure.NewJpegMediaParser()
			intfc, err := jmp.ParseBytes(data)
			if err != nil {
				t.Fatalf("Failed to parse JPEG: %v", err)
			}
			sl := intfc.(*jpegstructure.SegmentList)

			got := getJPEGOrientation(sl)
			if got != tt.expected {
				t.Errorf("expected orientation %d, got %d", tt.expected, got)
			}
		})
	}

	t.Run("no EXIF", func(t *testing.T) {
		data := createTestJPEG()
		jmp := jpegstructure.NewJpegMediaParser()
		intfc, err := jmp.ParseBytes(data)
		if err != nil {
			t.Fatalf("Failed to parse JPEG: %v", err)
		}
		sl := intfc.(*jpegstructure.SegmentList)

		got := getJPEGOrientation(sl)
		if got != 1 {
			t.Errorf("expected orientation 1 for no-EXIF, got %d", got)
		}
	})
}

func TestApplyOrientation(t *testing.T) {
	// Create a non-square JPEG (20x10) to verify dimension changes
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	data := buf.Bytes()

	tests := []struct {
		name           string
		orientation    int
		expectedWidth  int
		expectedHeight int
	}{
		{"orientation 1 - no change", 1, 20, 10},
		{"orientation 2 - flip H", 2, 20, 10},
		{"orientation 3 - rotate 180", 3, 20, 10},
		{"orientation 4 - flip V", 4, 20, 10},
		{"orientation 5 - transpose", 5, 10, 20},
		{"orientation 6 - rotate 90 CW", 6, 10, 20},
		{"orientation 7 - transverse", 7, 10, 20},
		{"orientation 8 - rotate 90 CCW", 8, 10, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corrected, err := applyOrientation(data, tt.orientation)
			if err != nil {
				t.Fatalf("applyOrientation failed: %v", err)
			}

			decoded, err := jpeg.Decode(bytes.NewReader(corrected))
			if err != nil {
				t.Fatalf("Cannot decode corrected JPEG: %v", err)
			}

			bounds := decoded.Bounds()
			if bounds.Dx() != tt.expectedWidth || bounds.Dy() != tt.expectedHeight {
				t.Errorf("Expected dimensions %dx%d, got %dx%d",
					tt.expectedWidth, tt.expectedHeight,
					bounds.Dx(), bounds.Dy())
			}
		})
	}

	// Orientation 1 should return data unchanged
	unchanged, err := applyOrientation(data, 1)
	if err != nil {
		t.Fatalf("applyOrientation(1) failed: %v", err)
	}
	if !bytes.Equal(unchanged, data) {
		t.Error("Orientation 1 should return data unchanged")
	}
}

func TestStripMetadata_JPEG_OrientationCorrected(t *testing.T) {
	tests := []struct {
		name           string
		orientation    uint16
		expectedWidth  int
		expectedHeight int
	}{
		{"orientation 1 - normal", 1, 20, 10},
		{"orientation 2 - flip H", 2, 20, 10},
		{"orientation 3 - rotate 180", 3, 20, 10},
		{"orientation 4 - flip V", 4, 20, 10},
		{"orientation 5 - transpose", 5, 10, 20},
		{"orientation 6 - rotate 90 CW", 6, 10, 20},
		{"orientation 7 - transverse", 7, 10, 20},
		{"orientation 8 - rotate 90 CCW", 8, 10, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := createTestJPEGWithOrientation(tt.orientation)

			stripped, err := StripMetadata(data, "image/jpeg")
			if err != nil {
				t.Fatalf("StripMetadata failed: %v", err)
			}

			// Verify still valid JPEG
			if !bytes.HasPrefix(stripped, []byte{0xFF, 0xD8, 0xFF}) {
				t.Error("Output is not a valid JPEG")
			}

			// Decode and verify dimensions
			decoded, err := jpeg.Decode(bytes.NewReader(stripped))
			if err != nil {
				t.Fatalf("Cannot decode output JPEG: %v", err)
			}

			bounds := decoded.Bounds()
			if bounds.Dx() != tt.expectedWidth || bounds.Dy() != tt.expectedHeight {
				t.Errorf("Expected dimensions %dx%d, got %dx%d",
					tt.expectedWidth, tt.expectedHeight,
					bounds.Dx(), bounds.Dy())
			}

			// Verify EXIF is stripped
			jmp := jpegstructure.NewJpegMediaParser()
			intfc, err := jmp.ParseBytes(stripped)
			if err != nil {
				t.Fatalf("Cannot parse output JPEG: %v", err)
			}
			sl := intfc.(*jpegstructure.SegmentList)
			_, _, _, findErr := sl.DumpExif()
			if findErr == nil {
				t.Error("EXIF data should have been stripped but was still present")
			}
		})
	}
}

func TestStripMetadata_JPEG_NoEXIF(t *testing.T) {
	data := createTestJPEG()

	stripped, err := StripMetadata(data, "image/jpeg")
	if err != nil {
		t.Fatalf("StripMetadata failed: %v", err)
	}

	_, err = jpeg.Decode(bytes.NewReader(stripped))
	if err != nil {
		t.Fatalf("Cannot decode output JPEG: %v", err)
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
