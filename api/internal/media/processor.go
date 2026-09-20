package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

var (
	ErrImageTooLarge    = errors.New("image dimensions exceed the safety limit")
	ErrUnsupportedImage = errors.New("image is not a supported raster format")
	ErrUnknownVariant   = errors.New("unknown media variant")
)

const (
	DerivativeVersion uint32 = 2
	DerivativeType           = "image/png"
)

type Limits struct {
	MaxDimension int
	MaxPixels    int64
}

func DefaultLimits() Limits {
	return Limits{MaxDimension: 16_384, MaxPixels: 40_000_000}
}

type Variant struct {
	Name      string
	MaxWidth  int
	MaxHeight int
	Blurred   bool
}

var variants = []Variant{
	{Name: "grid", MaxWidth: 640, MaxHeight: 640},
	{Name: "grid_blurred", MaxWidth: 640, MaxHeight: 640, Blurred: true},
	{Name: "detail", MaxWidth: 1600, MaxHeight: 1600},
	{Name: "detail_blurred", MaxWidth: 1600, MaxHeight: 1600, Blurred: true},
	{Name: "thumb", MaxWidth: 160, MaxHeight: 160},
	{Name: "thumb_blurred", MaxWidth: 160, MaxHeight: 160, Blurred: true},
}

var linkCards = []Variant{
	{Name: "og", MaxWidth: 1200, MaxHeight: 630},
	{Name: "og_blurred", MaxWidth: 1200, MaxHeight: 630, Blurred: true},
}

var previewField = color.RGBA{R: 0x05, G: 0x05, B: 0x05, A: 0xff}

func LinkCardByName(name string) (Variant, bool) {
	for _, preview := range linkCards {
		if preview.Name == name {
			return preview, true
		}
	}
	return Variant{}, false
}

func VariantNames() []string {
	names := make([]string, 0, len(variants))
	for _, variant := range variants {
		names = append(names, variant.Name)
	}
	return names
}

func VariantByName(name string) (Variant, bool) {
	for _, variant := range variants {
		if variant.Name == name {
			return variant, true
		}
	}
	return Variant{}, false
}

type Derivative struct {
	Variant string
	Bytes   []byte
}

type Prepared struct {
	Width       int
	Height      int
	Derivatives []Derivative
}

type Processor struct {
	limits  Limits
	encoder Encoder
}

func NewProcessor(limits Limits) *Processor {
	return NewProcessorWithEncoder(limits, PNGEncoder{})
}

type Encoder interface {
	MediaType() string
	Encode(io.Writer, image.Image) error
}

type PNGEncoder struct{}

func (PNGEncoder) MediaType() string { return DerivativeType }

func (PNGEncoder) Encode(w io.Writer, source image.Image) error {
	return png.Encode(w, source)
}

func NewProcessorWithEncoder(limits Limits, encoder Encoder) *Processor {
	return &Processor{limits: limits, encoder: encoder}
}

func (p *Processor) DerivativeType() string {
	return p.encoder.MediaType()
}

func (p *Processor) Prepare(ctx context.Context, source io.Reader) (Prepared, error) {
	decoded, width, height, err := p.decode(ctx, source)
	if err != nil {
		return Prepared{}, err
	}
	prepared := Prepared{
		Width:       width,
		Height:      height,
		Derivatives: make([]Derivative, 0, len(variants)),
	}
	for _, variant := range variants {
		if err := ctx.Err(); err != nil {
			return Prepared{}, err
		}
		derivative, err := p.render(decoded, variant)
		if err != nil {
			return Prepared{}, fmt.Errorf("render %s variant: %w", variant.Name, err)
		}
		prepared.Derivatives = append(prepared.Derivatives, derivative)
	}
	return prepared, nil
}

func (p *Processor) Render(ctx context.Context, source io.Reader, name string) (Derivative, error) {
	variant, ok := VariantByName(name)
	if !ok {
		return Derivative{}, ErrUnknownVariant
	}
	decoded, _, _, err := p.decode(ctx, source)
	if err != nil {
		return Derivative{}, err
	}
	if err := ctx.Err(); err != nil {
		return Derivative{}, err
	}
	return p.render(decoded, variant)
}

func (p *Processor) ComposeLinkCard(
	ctx context.Context,
	source io.Reader,
	name string,
) (Derivative, error) {
	preview, ok := LinkCardByName(name)
	if !ok {
		return Derivative{}, ErrUnknownVariant
	}
	decoded, _, _, err := p.decode(ctx, source)
	if err != nil {
		return Derivative{}, err
	}
	if err := ctx.Err(); err != nil {
		return Derivative{}, err
	}
	canvas := image.NewRGBA(image.Rect(0, 0, preview.MaxWidth, preview.MaxHeight))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: previewField}, image.Point{}, draw.Src)
	if preview.Blurred {
		decoded = obscure(decoded)
	}
	sourceSize := decoded.Bounds().Size()
	width, height := boundedSize(sourceSize.X, sourceSize.Y, preview.MaxWidth, preview.MaxHeight)
	left := (preview.MaxWidth - width) / 2
	top := (preview.MaxHeight - height) / 2
	draw.CatmullRom.Scale(
		canvas, image.Rect(left, top, left+width, top+height),
		decoded, decoded.Bounds(), draw.Over, nil,
	)
	return p.encode(canvas, preview.Name)
}

func (p *Processor) decode(ctx context.Context, source io.Reader) (image.Image, int, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, 0, err
	}
	encoded, err := io.ReadAll(source)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("read image: %w", err)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(encoded))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("read image dimensions: %w", ErrUnsupportedImage)
	}
	if err := p.checkDimensions(config.Width, config.Height); err != nil {
		return nil, 0, 0, err
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, 0, err
	}
	decoded, _, err := image.Decode(bytes.NewReader(encoded))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode image: %w", ErrUnsupportedImage)
	}
	size := decoded.Bounds().Size()
	if size.X != config.Width || size.Y != config.Height {
		return nil, 0, 0, fmt.Errorf("decoded dimensions changed from %dx%d to %dx%d: %w",
			config.Width, config.Height, size.X, size.Y, ErrUnsupportedImage)
	}
	return decoded, config.Width, config.Height, nil
}

func (p *Processor) checkDimensions(width, height int) error {
	if width <= 0 || height <= 0 || width > p.limits.MaxDimension || height > p.limits.MaxDimension {
		return fmt.Errorf("image is %dx%d: %w", width, height, ErrImageTooLarge)
	}
	if int64(width) > p.limits.MaxPixels/int64(height) {
		return fmt.Errorf("image is %dx%d: %w", width, height, ErrImageTooLarge)
	}
	return nil
}

func (p *Processor) render(source image.Image, variant Variant) (Derivative, error) {
	sourceSize := source.Bounds().Size()
	width, height := boundedSize(sourceSize.X, sourceSize.Y, variant.MaxWidth, variant.MaxHeight)
	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(resized, resized.Bounds(), source, source.Bounds(), draw.Over, nil)
	output := image.Image(resized)
	if variant.Blurred {
		output = obscure(resized)
	}
	return p.encode(output, variant.Name)
}

func (p *Processor) encode(source image.Image, variant string) (Derivative, error) {
	var encoded bytes.Buffer
	if err := p.encoder.Encode(&encoded, source); err != nil {
		return Derivative{}, fmt.Errorf("encode %s: %w", p.encoder.MediaType(), err)
	}
	return Derivative{Variant: variant, Bytes: encoded.Bytes()}, nil
}

func boundedSize(width, height, maxWidth, maxHeight int) (int, int) {
	scale := math.Min(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height))
	if scale >= 1 {
		return width, height
	}
	return max(1, int(math.Round(float64(width)*scale))),
		max(1, int(math.Round(float64(height)*scale)))
}

func obscure(source image.Image) image.Image {
	size := source.Bounds().Size()
	if size.X == 0 || size.Y == 0 {
		return source
	}

	current := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	draw.Draw(current, current.Bounds(), source, source.Bounds().Min, draw.Src)
	minimum := min(size.X, size.Y)
	sigma := math.Min(72, math.Max(4, float64(minimum)*0.045))
	if minimum < 12 {
		sigma = math.Max(0.8, float64(minimum)*0.25)
	}
	for _, radius := range gaussianBoxRadii(sigma, 3) {
		current = boxBlurRGBA(current, radius)
	}
	return current
}

// gaussianBoxRadii returns the box sizes for a close three-pass Gaussian
// approximation. Sliding box passes keep large source images inexpensive while
// producing a smooth field rather than visible censorship tiles.
func gaussianBoxRadii(sigma float64, passes int) []int {
	ideal := math.Sqrt((12 * sigma * sigma / float64(passes)) + 1)
	lower := int(math.Floor(ideal))
	if lower%2 == 0 {
		lower--
	}
	upper := lower + 2
	numerator := 12*sigma*sigma - float64(passes*lower*lower+4*passes*lower+3*passes)
	useLower := int(math.Round(numerator / float64(-4*lower-4)))
	useLower = max(0, min(passes, useLower))

	radii := make([]int, passes)
	for index := range radii {
		width := upper
		if index < useLower {
			width = lower
		}
		radii[index] = max(0, (width-1)/2)
	}
	return radii
}

func boxBlurRGBA(source *image.RGBA, radius int) *image.RGBA {
	if radius < 1 {
		copy := image.NewRGBA(source.Bounds())
		draw.Draw(copy, copy.Bounds(), source, source.Bounds().Min, draw.Src)
		return copy
	}
	horizontal := image.NewRGBA(source.Bounds())
	boxBlurHorizontal(horizontal, source, radius)
	vertical := image.NewRGBA(source.Bounds())
	boxBlurVertical(vertical, horizontal, radius)
	return vertical
}

func boxBlurHorizontal(target, source *image.RGBA, radius int) {
	bounds := source.Bounds()
	divisor := 2*radius + 1
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		var red, green, blue, alpha int
		for offset := -radius; offset <= radius; offset++ {
			pixel := source.RGBAAt(clamp(bounds.Min.X+offset, bounds.Min.X, bounds.Max.X-1), y)
			red += int(pixel.R)
			green += int(pixel.G)
			blue += int(pixel.B)
			alpha += int(pixel.A)
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			target.SetRGBA(x, y, color.RGBA{
				R: uint8(red / divisor), G: uint8(green / divisor),
				B: uint8(blue / divisor), A: uint8(alpha / divisor),
			})
			leaving := source.RGBAAt(clamp(x-radius, bounds.Min.X, bounds.Max.X-1), y)
			entering := source.RGBAAt(clamp(x+radius+1, bounds.Min.X, bounds.Max.X-1), y)
			red += int(entering.R) - int(leaving.R)
			green += int(entering.G) - int(leaving.G)
			blue += int(entering.B) - int(leaving.B)
			alpha += int(entering.A) - int(leaving.A)
		}
	}
}

func boxBlurVertical(target, source *image.RGBA, radius int) {
	bounds := source.Bounds()
	divisor := 2*radius + 1
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		var red, green, blue, alpha int
		for offset := -radius; offset <= radius; offset++ {
			pixel := source.RGBAAt(x, clamp(bounds.Min.Y+offset, bounds.Min.Y, bounds.Max.Y-1))
			red += int(pixel.R)
			green += int(pixel.G)
			blue += int(pixel.B)
			alpha += int(pixel.A)
		}
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			target.SetRGBA(x, y, color.RGBA{
				R: uint8(red / divisor), G: uint8(green / divisor),
				B: uint8(blue / divisor), A: uint8(alpha / divisor),
			})
			leaving := source.RGBAAt(x, clamp(y-radius, bounds.Min.Y, bounds.Max.Y-1))
			entering := source.RGBAAt(x, clamp(y+radius+1, bounds.Min.Y, bounds.Max.Y-1))
			red += int(entering.R) - int(leaving.R)
			green += int(entering.G) - int(leaving.G)
			blue += int(entering.B) - int(leaving.B)
			alpha += int(entering.A) - int(leaving.A)
		}
	}
}

func clamp(value, low, high int) int {
	return min(high, max(low, value))
}
