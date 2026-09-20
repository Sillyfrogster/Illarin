package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"slices"
	"testing"
)

func TestNamedImageSizesAreTheRelaunchSet(t *testing.T) {
	t.Parallel()
	want := []string{
		"grid", "grid_blurred",
		"detail", "detail_blurred",
		"thumb", "thumb_blurred",
	}
	if got := ImageSizeNames(); !slices.Equal(got, want) {
		t.Fatalf("imageSizes = %v, want %v", got, want)
	}
	if _, ok := ImageSizeByName("og"); ok {
		t.Fatal("og was accepted as an ordinary media size")
	}
	if ImageSizeVersion != 2 {
		t.Fatal("the smooth blur must not reuse the mosaic rendered cache")
	}
}

func TestAnImageSizeBoundsWithoutCroppingOrUpscaling(t *testing.T) {
	t.Parallel()
	source := image.NewRGBA(image.Rect(0, 0, 1200, 600))
	encoded := encodePNG(t, source)

	prepared, err := NewProcessor(DefaultLimits()).Prepare(
		context.Background(), bytes.NewReader(encoded),
	)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if prepared.Width != 1200 || prepared.Height != 600 {
		t.Fatalf("native dimensions = %dx%d, want 1200x600", prepared.Width, prepared.Height)
	}

	grid := sizeNamed(t, prepared.Sizes, "grid")
	decoded, err := png.Decode(bytes.NewReader(grid.Bytes))
	if err != nil {
		t.Fatalf("decode grid rendered: %v", err)
	}
	if got := decoded.Bounds().Size(); got.X != 640 || got.Y != 320 {
		t.Fatalf("grid dimensions = %dx%d, want 640x320", got.X, got.Y)
	}

	small := image.NewRGBA(image.Rect(0, 0, 80, 40))
	prepared, err = NewProcessor(DefaultLimits()).Prepare(
		context.Background(), bytes.NewReader(encodePNG(t, small)),
	)
	if err != nil {
		t.Fatalf("Prepare small image: %v", err)
	}
	thumb := sizeNamed(t, prepared.Sizes, "thumb")
	decoded, err = png.Decode(bytes.NewReader(thumb.Bytes))
	if err != nil {
		t.Fatalf("decode thumb rendered: %v", err)
	}
	if got := decoded.Bounds().Size(); got.X != 80 || got.Y != 40 {
		t.Fatalf("small thumb dimensions = %dx%d, want native 80x40", got.X, got.Y)
	}
}

func TestOversizedHeaderIsRejectedBeforePixelDecode(t *testing.T) {
	t.Parallel()
	limits := DefaultLimits()
	_, err := NewProcessor(limits).Prepare(
		context.Background(), bytes.NewReader(pngHeader(40_000, 40_000)),
	)
	if !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("Prepare error = %v, want ErrImageTooLarge", err)
	}
}

func TestBlurredCounterpartDoesNotCarryClearPixels(t *testing.T) {
	t.Parallel()
	source := image.NewRGBA(image.Rect(0, 0, 640, 320))
	for y := range 320 {
		for x := range 640 {
			if x < 320 {
				source.Set(x, y, color.Black)
			} else {
				source.Set(x, y, color.White)
			}
		}
	}
	prepared, err := NewProcessor(DefaultLimits()).Prepare(
		context.Background(), bytes.NewReader(encodePNG(t, source)),
	)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	clear := sizeNamed(t, prepared.Sizes, "grid")
	blurred := sizeNamed(t, prepared.Sizes, "grid_blurred")
	if bytes.Equal(clear.Bytes, blurred.Bytes) {
		t.Fatal("blurred rendered is byte-identical to the clear rendered")
	}
	decoded, err := png.Decode(bytes.NewReader(blurred.Bytes))
	if err != nil {
		t.Fatalf("decode blurred rendered: %v", err)
	}
	levels := make(map[uint32]struct{})
	for x := 240; x < 400; x++ {
		red, _, _, _ := decoded.At(x, 160).RGBA()
		levels[red] = struct{}{}
	}
	if len(levels) < 32 {
		t.Fatalf("blur transition has %d colour levels, want a smooth field rather than mosaic blocks", len(levels))
	}
}

func TestEncoderCanBeReplaced(t *testing.T) {
	t.Parallel()
	encoder := &recordingEncoder{}
	processor := NewProcessorWithEncoder(DefaultLimits(), encoder)
	prepared, err := processor.Prepare(
		context.Background(), bytes.NewReader(encodePNG(t, image.NewRGBA(image.Rect(0, 0, 2, 2)))),
	)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if encoder.calls != len(ImageSizeNames()) {
		t.Fatalf("encoder calls = %d, want %d", encoder.calls, len(ImageSizeNames()))
	}
	for _, rendered := range prepared.Sizes {
		if string(rendered.Bytes) != "replacement encoding" {
			t.Fatalf("%s bytes = %q", rendered.Size, rendered.Bytes)
		}
	}
	if processor.ImageSizeMediaType() != "image/example" {
		t.Fatalf("rendered type = %q", processor.ImageSizeMediaType())
	}
}

func TestOGIsASeparateComposedPreview(t *testing.T) {
	t.Parallel()
	source := image.NewRGBA(image.Rect(0, 0, 100, 200))
	for y := range 200 {
		for x := range 100 {
			source.Set(x, y, color.RGBA{R: 12, G: 34, B: 56, A: 255})
		}
	}
	processor := NewProcessor(DefaultLimits())
	preview, err := processor.ComposeLinkCard(
		context.Background(), bytes.NewReader(encodePNG(t, source)), "og",
	)
	if err != nil {
		t.Fatalf("ComposeLinkCard: %v", err)
	}
	if preview.Size != "og" {
		t.Fatalf("preview size = %q, want og", preview.Size)
	}
	decoded, err := png.Decode(bytes.NewReader(preview.Bytes))
	if err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if got := decoded.Bounds().Size(); got.X != 1200 || got.Y != 630 {
		t.Fatalf("preview dimensions = %dx%d, want 1200x630", got.X, got.Y)
	}
	if got := color.RGBAModel.Convert(decoded.At(0, 0)).(color.RGBA); got != previewField {
		t.Fatalf("preview corner = %#v, want the carbon field", got)
	}
	if _, ordinary := ImageSizeByName("og"); ordinary {
		t.Fatal("composed og preview entered the ordinary size set")
	}
}

type recordingEncoder struct {
	calls int
}

func (e *recordingEncoder) MediaType() string { return "image/example" }

func (e *recordingEncoder) Encode(w io.Writer, _ image.Image) error {
	e.calls++
	_, err := io.WriteString(w, "replacement encoding")
	return err
}

func sizeNamed(t *testing.T, sizes []Rendered, name string) Rendered {
	t.Helper()
	for _, rendered := range sizes {
		if rendered.Size == name {
			return rendered
		}
	}
	t.Fatalf("rendered %q is missing", name)
	return Rendered{}
}

func encodePNG(t *testing.T, source image.Image) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return encoded.Bytes()
}

func pngHeader(width, height uint32) []byte {
	var out bytes.Buffer
	out.Write([]byte("\x89PNG\r\n\x1a\n"))
	data := make([]byte, 13)
	binary.BigEndian.PutUint32(data[0:4], width)
	binary.BigEndian.PutUint32(data[4:8], height)
	data[8] = 8
	data[9] = 6
	writePNGChunk(&out, "IHDR", data)
	writePNGChunk(&out, "IEND", nil)
	return out.Bytes()
}

func writePNGChunk(out *bytes.Buffer, chunkType string, data []byte) {
	_ = binary.Write(out, binary.BigEndian, uint32(len(data)))
	out.WriteString(chunkType)
	out.Write(data)
	checksum := crc32.NewIEEE()
	_, _ = checksum.Write([]byte(chunkType))
	_, _ = checksum.Write(data)
	_ = binary.Write(out, binary.BigEndian, checksum.Sum32())
}

func TestSocialPreviewOfFlaggedWorkIsBlurred(t *testing.T) {
	t.Parallel()
	source := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for y := range 400 {
		for x := range 400 {
			if x < 200 {
				source.Set(x, y, color.Black)
			} else {
				source.Set(x, y, color.White)
			}
		}
	}
	encoded := encodePNG(t, source)
	processor := NewProcessor(DefaultLimits())

	clear, err := processor.ComposeLinkCard(
		context.Background(), bytes.NewReader(encoded), "og",
	)
	if err != nil {
		t.Fatalf("compose og: %v", err)
	}
	blurred, err := processor.ComposeLinkCard(
		context.Background(), bytes.NewReader(encoded), "og_blurred",
	)
	if err != nil {
		t.Fatalf("compose og_blurred: %v", err)
	}

	if blurred.Size != "og_blurred" {
		t.Fatalf("preview size = %q, want og_blurred", blurred.Size)
	}
	if bytes.Equal(clear.Bytes, blurred.Bytes) {
		t.Fatal("the blurred social preview is byte-identical to the clear one")
	}
	decoded, err := png.Decode(bytes.NewReader(blurred.Bytes))
	if err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if got := decoded.Bounds().Size(); got.X != 1200 || got.Y != 630 {
		t.Fatalf("preview dimensions = %dx%d, want 1200x630", got.X, got.Y)
	}
	if _, ordinary := ImageSizeByName("og_blurred"); ordinary {
		t.Fatal("the blurred composed preview entered the ordinary size set")
	}
	if _, ok := LinkCardByName("grid"); ok {
		t.Fatal("an ordinary size was accepted as a social preview")
	}
}
