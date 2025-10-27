package imacon

// The Imacon scene defines an extendible canvas that contains 3 main building blocks for composing image context:
// 1. Text - Text Labels
// 2. Image - Image, infographics that gives visual context
// 3. Pane - A container that holds Texts and Images, with layout properties such as padding, margin. Support object alignment within the pane. Support auto-tiling of objects to match the best output size efficiency.

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/fogleman/gg"
)

type Engine struct {
	cfg Config
}

type Config struct {
	MaxCanvasWidth  int         `json:"max_canvas_width" doc:"The maximum width of the canvas to compose images on."`
	MaxCanvasHeight int         `json:"max_canvas_height" doc:"The maximum height of the canvas to compose images on."`
	FgColor         color.Color `json:"fg_color" doc:"The foreground color used for text and shapes."`
	BgColor         color.Color `json:"bg_color" doc:"The background color of the canvas."`
}

func New(cfg Config) *Engine {
	return &Engine{cfg: cfg}
}

type Canvas struct {
	Width  int
	Height int
	Raw    image.Image
}

func (c *Canvas) ToJpeg(writer io.Writer, options *jpeg.Options) error {
	if err := jpeg.Encode(writer, c.Raw, options); err != nil {
		return err
	}
	return nil
}

func (c *Canvas) ToPng(writer io.Writer) error {
	if err := png.Encode(writer, c.Raw); err != nil {
		return err
	}
	return nil
}

func (e *Engine) Render(scene *Scene) (*Canvas, error) {

	// NOTE: currently we test using hardcoded size
	width := e.cfg.MaxCanvasWidth
	height := e.cfg.MaxCanvasHeight
	bgColor := e.cfg.BgColor
	if bgColor == nil {
		bgColor = color.White
	}
	fgColor := e.cfg.FgColor
	if fgColor == nil {
		fgColor = color.Black
	}

	ctx := gg.NewContext(width, height)
	ctx.SetColor(bgColor)
	ctx.Fill()
	ctx.Clear()

	ctx.SetColor(fgColor)
	pane := scene.Main
	pane.Draw(ctx)
	canvas := &Canvas{
		Width:  width,
		Height: height,
		Raw:    ctx.Image(),
	}

	return canvas, nil
}

type Scene struct {
	Main *Pane // The main pane that holds all the objects to be rendered.
	// Expect there are some layout properties here in the future
	// ...
}

type Pane struct {
	Objects []Tileable // The objects within the pane, which can be TextBlocks or ImageBlocks
}

func (p *Pane) Draw(ctx *gg.Context) {
	for _, obj := range p.Objects {
		obj.Draw(ctx)
	}
}

type TextBlock struct {
	// Representation of a plain-text.
	Text string
}

func (t *TextBlock) Draw(ctx *gg.Context) {
	ctx.DrawString(t.Text, 0, 0)
}

type ImageBlock struct {
	// Representation of an image, with a custom label for identification.
	Raw   []byte
	Label string
}

type Drawable interface {
	// Drawable defines the behavior of objects that can be drawn onto the scene.
	Draw(ctx *gg.Context)
}

type Tileable interface {
	// Tilable specific the tiling behavior of the pane within the window
	Drawable
}

func NewScene(main *Pane) *Scene {
	return &Scene{Main: main}
}
