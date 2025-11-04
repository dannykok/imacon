package imacon

// The Imacon scene defines an extendible canvas that contains 3 main building blocks for composing image context:
// 1. Text - Text Labels
// 2. Image - Image, infographics that gives visual context
// 3. Pane - A container that holds Texts and Images, with layout properties such as padding, margin. Support object alignment within the pane. Support auto-tiling of objects to match the best output size efficiency.

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"math"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
)

//go:embed assets/fonts/JetBrainsMono-Regular.ttf
var embeddedFont embed.FS

const (
	DefaultMinPad      = 10.0 // The minimum padding between tiles and canvas edges
	DefaultLineSpacing = 1.5  // The default line spacing for text rendering
	DefaultLabelPad    = 3.0  // The default padding between image and its label
	DefaultMinFontSize = 12.0 // The default minimum font size
	DefaultMaxFontSize = 32.0 // The default maximum font size
)

type Engine struct {
	cfg Config
}

// Drawable defines the behavior of objects that can be drawn onto the scene.
type Drawable interface {
	Draw(ctx *gg.Context, cw float64, ch float64) // Draw the object onto the given context with specified canvas width and height
}

// Tilable specific the tiling behavior of the pane within the window
type Tileable interface {
	Drawable
	IntrinsicSize(ctx *gg.Context, expectedWidth float64, expectedHeight float64) (float64, float64) // return the intrinsic width and height of the object
}

// The configuration options for the Imacon rendering engine.
type Config struct {
	MaxCanvasWidth  int         // The maximum width of the canvas to compose images on.
	MaxCanvasHeight int         // The maximum height of the canvas to compose images on.
	FgColor         color.Color // The foreground color used for text and shapes.
	BgColor         color.Color // The background color of the canvas.
	FontSize        float64     // The default font size for text rendering.
}

func New(cfg Config) *Engine {
	return &Engine{cfg: cfg}
}

// Canvas represents the rendered image canvas.
type Canvas struct {
	Width  int         // The width of the canvas in pixels.
	Height int         // The height of the canvas in pixels.
	Raw    image.Image // The raw image data of the canvas.
}

// ToJpeg encodes the canvas image to JPEG format and writes it to the provided writer.
func (c *Canvas) ToJpeg(writer io.Writer, options *jpeg.Options) error {
	if err := jpeg.Encode(writer, c.Raw, options); err != nil {
		return err
	}
	return nil
}

// ToPng encodes the canvas image to PNG format and writes it to the provided writer.
func (c *Canvas) ToPng(writer io.Writer) error {
	if err := png.Encode(writer, c.Raw); err != nil {
		return err
	}
	return nil
}

// Render generates a canvas by rendering the provided scene according to the engine's configuration.
func (e *Engine) Render(scene *Scene) (*Canvas, error) {
	// NOTE: currently we test using hardcoded size
	width := e.cfg.MaxCanvasWidth
	height := e.cfg.MaxCanvasHeight

	// define config values
	bgColor := e.cfg.BgColor
	if bgColor == nil {
		bgColor = color.White
	}
	fgColor := e.cfg.FgColor
	if fgColor == nil {
		fgColor = color.Black
	}
	fontSize := e.cfg.FontSize
	if fontSize == 0 {
		fontSize = 12
	}

	ctx := gg.NewContext(width, height)
	ctx.SetColor(bgColor)
	ctx.Clear()

	ctx.SetColor(fgColor)

	fontData, err := embeddedFont.ReadFile("assets/fonts/JetBrainsMono-Regular.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded font: %w", err)
	}

	f, err := truetype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	face := truetype.NewFace(f, &truetype.Options{
		Size: fontSize,
		DPI:  72,
	})
	ctx.SetFontFace(face)

	pane := scene.Main
	pane.Draw(ctx, float64(width), float64(height))
	canvas := &Canvas{
		Width:  width,
		Height: height,
		Raw:    ctx.Image(),
	}

	return canvas, nil
}

// Scene represents the overall image composition, containing panes and their layout properties.
type Scene struct {
	Main *Pane // The main pane that holds all the objects to be rendered.
	// Expect there are some layout properties here in the future
	// ...
}

// Pane represents a container that holds multiple tileable objects (TextBlocks or ImageBlocks) and manages their layout.
type Pane struct {
	Objects []Tileable // The objects within the pane, which can be TextBlocks or ImageBlocks
}

// Shape stores the column count in each row, and the derived padding between tiles and canvas edges (per row)
// e.g. dim = [3,2,4] means the shape has 3 rows, with 3 columns in the first row, 2 in the second, and 4 in the third.
type Shape struct {
	Dim    []int
	ColPad []float64 // Column padding per row
}

// Return the shape of the tiling objects with a given canvas width and height
func (p *Pane) Shape(ctx *gg.Context, cw float64, ch float64, minPad float64) Shape {
	// currently we ignore height constraint
	dim := []int{}      // dimension
	cPad := []float64{} // column padding per row

	minPad = math.Max(minPad, DefaultMinPad)
	accWidth := 0.0
	colCount := 0
	for _, obj := range p.Objects {
		// let w, h be the tile size
		w, _ := obj.IntrinsicSize(ctx, cw, 0)
		widthDemand := accWidth + w + float64(colCount+2)*minPad
		if widthDemand > cw {
			if colCount == 0 {
				// even a single tile cannot fit, force to add one column
				accWidth = w
				colCount = 1
				cPad = append(cPad, math.Max(math.Trunc((cw-w)/float64(colCount+1)), minPad))
				dim = append(dim, 1)
				accWidth = 0
				colCount = 0
			} else {
				// finish the current row
				cPad = append(cPad, math.Max(math.Trunc((cw-accWidth)/float64(colCount+1)), minPad))
				dim = append(dim, colCount)
				// start a new row
				accWidth = w
				colCount = 1
			}
		} else {
			accWidth += w
			colCount++
		}
	}
	if colCount > 0 {
		cPad = append(cPad, math.Max(math.Trunc((cw-accWidth)/float64(colCount+1)), minPad))
		dim = append(dim, colCount)
	}

	s := Shape{Dim: dim, ColPad: cPad}
	return s
}

func (p *Pane) Draw(ctx *gg.Context, cw float64, ch float64) {
	outerPad := DefaultMinPad
	ctx.Push()
	ctx.Translate(outerPad, outerPad)

	adjustedWidth := cw - 2*outerPad
	adjustedHeight := ch - 2*outerPad

	shape := p.Shape(ctx, adjustedWidth, adjustedHeight, DefaultMinPad)
	if shape.Dim == nil || shape.ColPad == nil {
		fmt.Println("Pane.Draw: invalid shape derived")
		ctx.Pop()
		return
	}

	itemIndex := 0
	rPad := DefaultMinPad
	for row, colCount := range shape.Dim {
		pad := shape.ColPad[row]
		objects := p.Objects[itemIndex : itemIndex+colCount]

		// we adopted a proportational scaling strategy, corresponding to object's intrinsic size
		// first calculate the total intrinsic width and height of the objects in this row
		totalW := 0.0

		for _, obj := range objects {
			w, _ := obj.IntrinsicSize(ctx, adjustedWidth, 0)
			totalW += w
		}

		// draw each object in this row
		itemIndex += colCount
		ctx.Push()
		ctx.Translate(pad, rPad)
		maxH := 0.0
		for _, obj := range objects {
			w, _ := obj.IntrinsicSize(ctx, adjustedWidth, 0)
			w = (w / totalW) * (adjustedWidth - float64(colCount+1)*pad)
			_, h := obj.IntrinsicSize(ctx, w, 0)
			obj.Draw(ctx, w, h)
			xtrans := math.Trunc(w + pad)
			ctx.Translate(xtrans, 0)
			if h > maxH {
				maxH = h
			}
		}
		ctx.Pop()
		if maxH == 0 {
			panic("Pane.Draw: invalid maxH calculated")
		}
		ctx.Translate(0, maxH+rPad)
	}
	ctx.Pop()
}

func (p *Pane) IntrinsicSize(ctx *gg.Context, expectedWidth float64, expectedHeight float64) (float64, float64) {
	outerPad := DefaultMinPad
	adjustedWidth := expectedWidth - 2*outerPad
	adjustedHeight := expectedHeight - 2*outerPad

	shape := p.Shape(ctx, adjustedWidth, adjustedHeight, DefaultMinPad)
	totalH := 0.0
	itemIndex := 0
	for row, colCount := range shape.Dim {
		pad := shape.ColPad[row]
		objects := p.Objects[itemIndex : itemIndex+colCount]
		itemIndex += colCount
		maxH := 0.0
		totalW := 0.0
		for _, obj := range objects {
			w, h := obj.IntrinsicSize(ctx, 0, 0)
			totalW += w
			if h > maxH {
				maxH = h
			}
		}
		scale := (adjustedWidth - float64(colCount+1)*pad) / totalW
		totalH += maxH*scale + DefaultMinPad
	}
	return expectedWidth, totalH + 2*outerPad
}

type TextBlockOpts struct {
	TextWrap bool // Whether to wrap text if it exceeds the pane width
}

type TextBlock struct {
	// Representation of a plain-text.
	Text string
	Opts TextBlockOpts
}

func NewTextBlock(text string, opts TextBlockOpts) *TextBlock {
	return &TextBlock{Text: text, Opts: opts}
}

func (t *TextBlock) Draw(ctx *gg.Context, cw float64, ch float64) {
	if t.Opts.TextWrap == false {
		ctx.DrawStringAnchored(t.Text, 0, 0, 0, 1)
	} else {
		maxWidth := float64(cw)
		ctx.DrawStringWrapped(t.Text, 0, 0, 0, 0, maxWidth, DefaultLineSpacing, gg.AlignLeft)
	}
}

func (t *TextBlock) IntrinsicSize(ctx *gg.Context, expectedWidth float64, expectedHeight float64) (float64, float64) {

	if expectedWidth == 0 {
		return ctx.MeasureMultilineString(t.Text, DefaultLineSpacing)
	} else {
		lines := ctx.WordWrap(t.Text, expectedWidth)
		maxWidth := 0.0
		for _, line := range lines {
			w, _ := ctx.MeasureString(line)
			if w > maxWidth {
				maxWidth = w
			}
		}
		totalHeight := float64(len(lines)) * ctx.FontHeight() * DefaultLineSpacing
		return maxWidth, totalHeight
	}
}

type ImageBlock struct {
	// Representation of an image, with a custom label for identification.
	Image image.Image
	Label *TextBlock
}

func NewImageBlock(file io.Reader, label string) *ImageBlock {
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("NewImageBlock: failed to load image from bytes:", err)
		return nil
	}
	textblock := NewTextBlock(label, TextBlockOpts{TextWrap: true})
	return &ImageBlock{Image: img, Label: textblock}
}

func (i *ImageBlock) Draw(ctx *gg.Context, cw float64, ch float64) {
	ctx.DrawImageAnchored(i.Image, 0, 0, 0, 0)
	ctx.Push()
	ctx.Translate(0, float64(i.Image.Bounds().Dy())+DefaultLabelPad)
	i.Label.Draw(ctx, cw, ch-float64(i.Image.Bounds().Dy()))
	ctx.Pop()
}

func (i *ImageBlock) IntrinsicSize(ctx *gg.Context, expectedWidth float64, expectedHeight float64) (float64, float64) {
	w := float64(i.Image.Bounds().Dx())
	h := float64(i.Image.Bounds().Dy())
	scale := 1.0
	if expectedWidth == 0 && expectedHeight == 0 {
		expectedWidth = w
	}
	if expectedWidth != 0 && expectedHeight == 0 {
		// only scale down image but not scale up
		if expectedWidth > w {
			expectedWidth = w
			scale = 1.0
		} else {
			scale = expectedWidth / w
		}
		_, textHeight := i.Label.IntrinsicSize(ctx, expectedWidth, 0)
		return expectedWidth, h*scale + textHeight + DefaultLabelPad
	} else if expectedWidth == 0 && expectedHeight != 0 {
		scale := expectedHeight / h
		newWidth := w * scale
		_, textHeight := i.Label.IntrinsicSize(ctx, newWidth, 0)
		return newWidth, expectedHeight + textHeight + DefaultLabelPad
	} else {
		// both width and height are defined, we scale based on the smaller scale factor to fit within the box
		scaleW := expectedWidth / w
		scaleH := expectedHeight / h
		scale := math.Min(scaleW, scaleH)
		newWidth := w * scale
		_, textHeight := i.Label.IntrinsicSize(ctx, newWidth, 0)
		return newWidth, h*scale + textHeight + DefaultLabelPad
	}
}

func NewScene(main *Pane) *Scene {
	return &Scene{Main: main}
}
