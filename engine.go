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
	DefaultOuterPad    = 24.0  // The default outer padding around the canvas
	DefaultMinPad      = 12.0  // The minimum padding between tiles
	DefaultLineSpacing = 1.5   // The default line spacing for text rendering
	DefaultLabelPad    = 3.0   // The default padding between image and its label
	DefaultMinFontSize = 12.0  // The default minimum font size
	DefaultMaxFontSize = 32.0  // The default maximum font size
	DefaultColWidth    = 720.0 // The default column width for tiling
	DefaultColPad      = 24.0  // The default padding between columns
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
	outerPad := DefaultOuterPad
	scale := 1.0

	fontData, err := embeddedFont.ReadFile("assets/fonts/JetBrainsMono-Regular.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded font: %w", err)
	}

	f, err := truetype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	fontFace := truetype.NewFace(f, &truetype.Options{
		Size: fontSize,
		DPI:  72,
	})

	// temp canvas to measure canvas size
	tempCtx := gg.NewContext(100, 100)
	tempCtx.SetFontFace(fontFace)
	width, height := scene.canvasSize(tempCtx, outerPad)

	// measure the scale factor used to fit within max canvas size
	if width > e.cfg.MaxCanvasWidth {
		scale = float64(e.cfg.MaxCanvasWidth) / float64(width)
		width = e.cfg.MaxCanvasWidth
	}
	if height > e.cfg.MaxCanvasHeight {
		scale = math.Min(scale, float64(e.cfg.MaxCanvasHeight)/float64(height))
		height = e.cfg.MaxCanvasHeight
	}

	ctx := gg.NewContext(width, height)
	ctx.SetColor(bgColor)
	ctx.Clear()
	ctx.ScaleAbout(scale, scale, 0, 0)
	ctx.SetColor(fgColor)

	ctx.SetFontFace(fontFace)
	ctx.Translate(outerPad, outerPad)

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

func (s *Scene) canvasSize(ctx *gg.Context, outerPad float64) (int, int) {
	if outerPad == 0 {
		outerPad = DefaultOuterPad
	}
	w, h := s.Main.IntrinsicSize(ctx, 0, 0)
	return int(w + outerPad*2), int(h + outerPad*2)
}

// Pane represents a container that holds multiple tileable objects (TextBlocks or ImageBlocks) and manages their layout.
type Pane struct {
	Objects      []Tileable // The objects within the pane, which can be TextBlocks or ImageBlocks
	PlannedShape *Shape     // The planned shape of the pane after layout calculation
	ColWidth     float64    // The fixed column width for tiling
	ColPad       float64    // The padding between columns
	RowPad       float64    // The padding between tiles in a column
}

func NewPane(objects []Tileable, colWidth float64, colPad float64, rowPad float64) *Pane {
	if colWidth == 0 {
		colWidth = DefaultColWidth
	}
	if colPad == 0 {
		colPad = DefaultColPad
	}
	if rowPad == 0 {
		rowPad = DefaultMinPad
	}
	return &Pane{
		Objects:  objects,
		ColWidth: colWidth,
		ColPad:   colPad,
		RowPad:   rowPad,
	}
}

func NewPaneWithShape(Shape *Shape, colWidth float64, colPad float64, rowPad float64) *Pane {
	if colWidth == 0 {
		colWidth = DefaultColWidth
	}
	if colPad == 0 {
		colPad = DefaultColPad
	}
	if rowPad == 0 {
		rowPad = DefaultMinPad
	}
	return &Pane{
		PlannedShape: Shape,
		ColWidth:     colWidth,
		ColPad:       colPad,
		RowPad:       rowPad,
	}
}

// Column represents a single column in the pane, containing multiple tileable objects. The column width will follow Pane.ColWidth when rendered.
type Column struct {
	Objects []Tileable
}

func (c Column) Height(ctx *gg.Context, colWidth float64, rowPad float64) float64 {
	totalH := 0.0
	for _, obj := range c.Objects {
		_, h := obj.IntrinsicSize(ctx, colWidth, 0)
		totalH += h
	}
	totalH += rowPad * float64(len(c.Objects)-1)
	return totalH
}

// Shape stores the layout shape of the pane in terms of columns and rows. It's a temporary view of underlying objects calculated using the greedy algorithm to fit into the best canvas size.
type Shape struct {
	Columns []Column // The columns in the pane
}

func NewShape(colCount int) *Shape {
	return &Shape{
		Columns: make([]Column, colCount),
	}
}

func NewShapeWithObjects(columns []Column) *Shape {
	return &Shape{
		Columns: columns,
	}
}

// TileProxy is a placeholder tileable object used for layout calculations, with pre-calculated size.
type TileProxy struct {
	Object Tileable // The actual tileable object being proxied
	Size   Size     // The pre-calculated size of the tile
}

func (t *TileProxy) Draw(ctx *gg.Context, cw float64, ch float64) {
	t.Object.Draw(ctx, cw, ch)
}

func (t *TileProxy) IntrinsicSize(ctx *gg.Context, expectedWidth float64, expectedHeight float64) (float64, float64) {
	return t.Size.Width, t.Size.Height
}

type Size struct {
	Width  float64
	Height float64
}

// Calculate and return the shape of the column layout of the pane.
// The algorithm finds the smallest footprint of canvas that can fit all objects in the pane.
func (p *Pane) Shape(ctx *gg.Context) (Shape, Size) {

	// we try to optimize the layout with the smallest bounding box, as well as lowest aspect ratio difference to 1:1
	maxCol := len(p.Objects) // maximum number of columns possible
	areaDotAr := math.MaxFloat64
	var bestShape *Shape
	var bestSize Size

	// Create proxies
	proxies := make([]Tileable, len(p.Objects))
	for i, obj := range p.Objects {
		w, h := obj.IntrinsicSize(ctx, p.ColWidth, 0)
		proxies[i] = &TileProxy{Object: obj, Size: Size{Width: w, Height: h}}
	}

	for colCount := 1; colCount <= maxCol; colCount++ {
		s := NewShape(colCount)
		deriveShape(ctx, s, proxies, p.ColWidth, p.RowPad)

		w, h := canvasSize(ctx, s, p.ColWidth, p.ColPad, p.RowPad)
		area := w * h
		ar := math.Max(w/h, h/w)
		if area*ar < areaDotAr {
			bestShape = s
			bestSize = Size{Width: w, Height: h}
			areaDotAr = area * ar
		}
	}

	return *bestShape, bestSize
}

// Greedy algorithm to push tiles into the shape's columns based on the given column width
func deriveShape(ctx *gg.Context, s *Shape, t []Tileable, colWidth float64, rowPad float64) {
	colCount := len(s.Columns)
	for _, tile := range t {
		minHeightCol := 0
		minHeight := math.MaxFloat64
		for colIndex := range colCount {
			h := s.Columns[colIndex].Height(ctx, colWidth, rowPad)
			if h < minHeight {
				minHeight = h
				minHeightCol = colIndex
			}
		}
		s.Columns[minHeightCol].Objects = append(s.Columns[minHeightCol].Objects, tile)
	}
}

// Calculate the canvas size based on the layout of given shape.
func canvasSize(ctx *gg.Context, shape *Shape, colWidth float64, colPad float64, rowPad float64) (float64, float64) {
	colCount := len(shape.Columns)
	totalW := float64(colCount)*colWidth + float64(colCount-1)*colPad
	maxH := 0.0
	for colIndex := range colCount {
		h := shape.Columns[colIndex].Height(ctx, colWidth, rowPad)
		if h > maxH {
			maxH = h
		}
	}
	return totalW, maxH
}

// Draw the pane onto the given context based on the provided shape.
func (p *Pane) DrawShape(ctx *gg.Context, shape Shape) {
	rPad := DefaultMinPad
	for colCount, column := range shape.Columns {
		ctx.Push()
		translateX := p.ColWidth*float64(colCount) + p.ColPad*float64(colCount)
		ctx.Translate(translateX, 0)
		for _, obj := range column.Objects {
			w, h := obj.IntrinsicSize(ctx, p.ColWidth, 0)
			obj.Draw(ctx, w, h)
			ctx.Translate(0, h+rPad)
		}
		ctx.Pop()
	}
}

func (p *Pane) Draw(ctx *gg.Context, cw float64, ch float64) {
	if p.PlannedShape != nil {
		p.DrawShape(ctx, *p.PlannedShape)
	} else {
		shape, _ := p.Shape(ctx)
		p.PlannedShape = &shape
		p.DrawShape(ctx, shape)
	}
}

func (p *Pane) IntrinsicSize(ctx *gg.Context, expectedWidth float64, expectedHeight float64) (float64, float64) {
	if p.PlannedShape != nil {
		return canvasSize(ctx, p.PlannedShape, p.ColWidth, p.ColPad, p.RowPad)
	} else {
		shape, size := p.Shape(ctx)
		p.PlannedShape = &shape
		return size.Width, size.Height
	}
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

func NewImageBlock(file io.Reader, label string) (*ImageBlock, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("NewImageBlock: failed to load image from bytes:", err)
		return nil, err
	}
	textblock := NewTextBlock(label, TextBlockOpts{TextWrap: true})
	return &ImageBlock{Image: img, Label: textblock}, nil
}

func (i *ImageBlock) Draw(ctx *gg.Context, cw float64, ch float64) {
	// scale down image if necessary
	ctx.Push()
	ctx.Push()
	scale := 1.0
	if float64(i.Image.Bounds().Dx()) > cw {
		scale = cw / float64(i.Image.Bounds().Dx())
		ctx.Scale(scale, scale)
	}
	ctx.DrawImageAnchored(i.Image, 0, 0, 0, 0)
	ctx.Pop()
	imageHeight := float64(i.Image.Bounds().Dy()) * scale
	ctx.Translate(0, imageHeight+DefaultLabelPad)
	i.Label.Draw(ctx, cw, ch-imageHeight-DefaultLabelPad)
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
