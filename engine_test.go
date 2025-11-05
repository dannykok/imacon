package imacon

import (
	"fmt"
	"os"
	"testing"

	"github.com/fogleman/gg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_IntrinsicSize(t *testing.T) {

	sampleTexts := []string{
		"Hello, World!",
		"A very long text block that is intended to test the text wrapping functionality within the Imacon engine. This text should be wrapped appropriately based on the specified width constraints to ensure that it fits well within the designated area without overflowing or being cut off.",
	}

	sampleImageBlocks := make([]*ImageBlock, 0)
	f, err := os.Open("assets/samples/sample_1.jpg") // 485x485
	require.NoError(t, err)
	defer f.Close()
	img, err := NewImageBlock(f, "Sample Image 1")
	require.NoError(t, err)
	sampleImageBlocks = append(sampleImageBlocks, img)

	f, err = os.Open("assets/samples/sample_2.jpg") // 819x1024
	require.NoError(t, err)
	defer f.Close()
	img, err = NewImageBlock(f, "Sample Image 2")
	require.NoError(t, err)
	sampleImageBlocks = append(sampleImageBlocks, img)

	ctx := gg.NewContext(1024, 1024)
	t.Run("TextBlock intrinsic size", func(t *testing.T) {
		for _, text := range sampleTexts {
			textBlock := NewTextBlock(text, TextBlockOpts{TextWrap: true})
			w, h := textBlock.IntrinsicSize(ctx, 300, 0)
			assert.Greater(t, w, 0.0, "Width should be greater than 0")
			assert.Greater(t, h, 0.0, "Height should be greater than 0")
			assert.LessOrEqual(t, w, 300.0, "Width should be less than or equal to 300 when wrapped")
		}
	})

	t.Run("ImageBlock intrinsic size", func(t *testing.T) {
		require.NoError(t, err)
		w, h := sampleImageBlocks[0].IntrinsicSize(ctx, 0, 0)
		assert.Equal(t, w, 485.0, "Width should be equal to image width")
		assert.Greater(t, h, 485.0, "height should be greater than image height")
	})

	t.Run("Pane intrinsic size (Text)", func(t *testing.T) {
		tb1 := NewTextBlock(sampleTexts[0], TextBlockOpts{TextWrap: true})
		tb2 := NewTextBlock(sampleTexts[1], TextBlockOpts{TextWrap: true})
		_, tb1H := tb1.IntrinsicSize(ctx, DefaultColWidth, 0)
		_, tb2H := tb2.IntrinsicSize(ctx, DefaultColWidth, 0)

		// create a single-column custom shape for testing
		shape := NewShape(1)
		shape.Columns[0].Objects = []Tileable{tb1, tb2}
		pane := NewPaneWithShape(shape, DefaultColWidth, DefaultColPad, DefaultMinPad)
		w, h := pane.IntrinsicSize(ctx, 0, 0)
		assert.Equal(t, DefaultColWidth, w, "Width should match pane width")
		assert.Equal(t, tb1H+tb2H+DefaultMinPad, h, "Height should be sum of text block heights and rPad")

		s := NewScene(pane)
		ctx := gg.NewContext(1024, 1024)
		sw, sh := s.canvasSize(ctx, DefaultOuterPad)
		assert.Equal(t, int(w+2*DefaultOuterPad), sw, "Scene width should match pane width plus outer padding")
		assert.Equal(t, int(h+2*DefaultOuterPad), sh, "Scene height should match pane height plus outer padding")
	})

	t.Run("Pane intrinsic size (Images)", func(t *testing.T) {
		ib1 := sampleImageBlocks[0]
		ib2 := sampleImageBlocks[1]
		_, ib1H := ib1.IntrinsicSize(ctx, DefaultColWidth, 0)
		_, ib2H := ib2.IntrinsicSize(ctx, DefaultColWidth, 0)

		// create a single-column custom shape for testing
		shape := NewShape(1)
		shape.Columns[0].Objects = []Tileable{ib1, ib2}
		pane := NewPaneWithShape(shape, DefaultColWidth, DefaultColPad, DefaultMinPad)
		w, h := pane.IntrinsicSize(ctx, 0, 0)
		assert.Equal(t, DefaultColWidth, w, "Width should match pane width")
		assert.Equal(t, ib1H+ib2H+DefaultMinPad, h, "Height should be sum of text block heights and rPad")

		s := NewScene(pane)
		ctx := gg.NewContext(1024, 1024)
		sw, sh := s.canvasSize(ctx, DefaultOuterPad)
		assert.Equal(t, int(w+2*DefaultOuterPad), sw, "Scene width should match pane width plus outer padding")
		assert.Equal(t, int(h+2*DefaultOuterPad), sh, "Scene height should match pane height plus outer padding")
	})
}

func Test_Render(t *testing.T) {
	// create test output folder
	_ = os.Mkdir("test_output", os.ModePerm)

	eng := New(Config{
		MaxCanvasWidth:  4096,
		MaxCanvasHeight: 4096,
		FontSize:        32,
	})

	sampleJpgBlocks := make([]Tileable, 10)
	for i := range sampleJpgBlocks {
		sampleJpgFile, err := os.Open("assets/samples/sample_1.jpg")
		require.NoError(t, err)
		defer sampleJpgFile.Close()
		jpg, err := NewImageBlock(sampleJpgFile, fmt.Sprintf("Sample %d", i+1))
		require.NoError(t, err)
		sampleJpgBlocks[i] = jpg
	}

	samplePngBlocks := make([]Tileable, 1)
	for i := range samplePngBlocks {
		samplePngFile, err := os.Open("assets/samples/glasses.png")
		require.NoError(t, err)
		defer samplePngFile.Close()
		png, err := NewImageBlock(samplePngFile, "Glasses")
		require.NoError(t, err)
		samplePngBlocks[i] = png
	}

	var sampleImageSets []Tileable

	sampleFile, err := os.Open("assets/samples/sample_1.jpg")
	require.NoError(t, err)
	defer sampleFile.Close()
	img, err := NewImageBlock(sampleFile, "Face")
	require.NoError(t, err)
	sampleImageSets = append(sampleImageSets, img)

	sampleFile, err = os.Open("assets/samples/sample_2.jpg")
	require.NoError(t, err)
	defer sampleFile.Close()
	img, err = NewImageBlock(sampleFile, "T-shirt")
	require.NoError(t, err)
	sampleImageSets = append(sampleImageSets, img)

	sampleFile, err = os.Open("assets/samples/sample_3.jpg")
	require.NoError(t, err)
	defer sampleFile.Close()
	img, err = NewImageBlock(sampleFile, "Necklace")
	require.NoError(t, err)
	sampleImageSets = append(sampleImageSets, img)

	sampleFile, err = os.Open("assets/samples/sample_4.jpg")
	require.NoError(t, err)
	defer sampleFile.Close()
	img, err = NewImageBlock(sampleFile, "Hair-style")
	require.NoError(t, err)
	sampleImageSets = append(sampleImageSets, img)

	testCases := []struct {
		name  string
		scene Scene
	}{
		{
			name: "Render Simple TextBlock",
			scene: Scene{
				Main: NewPane(
					[]Tileable{
						&TextBlock{Text: "Hello, World!"},
					}, 0, 0, 0),
			},
		},
		{
			name: "Render Paragraph",
			scene: Scene{
				Main: NewPane(
					[]Tileable{
						&TextBlock{Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", Opts: TextBlockOpts{TextWrap: true}},
					}, 0, 0, 0),
			},
		},
		{
			name: "Render multiple TextBlocks",
			scene: Scene{
				Main: NewPane(
					[]Tileable{
						&TextBlock{Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", Opts: TextBlockOpts{TextWrap: true}},
						&TextBlock{Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", Opts: TextBlockOpts{TextWrap: true}},
						&TextBlock{Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", Opts: TextBlockOpts{TextWrap: true}},
					}, 0, 0, 0),
			},
		},
		{
			name: "Render Simple ImageBlock",
			scene: Scene{
				Main: NewPane(sampleJpgBlocks[0:1], 0, 0, 0),
			},
		},
		{
			name: "Render Two ImageBlocks",
			scene: Scene{
				Main: NewPane(sampleJpgBlocks[0:2], 0, 0, 0),
			},
		},
		{
			name: "Render Many ImageBlocks",
			scene: Scene{
				Main: NewPane(sampleJpgBlocks[0:8], 0, 0, 0),
			},
		},
		{
			name: "Render Jpg and Png",
			scene: Scene{
				Main: NewPane(
					[]Tileable{
						NewPane(
							[]Tileable{
								&TextBlock{
									Text: "Imacon is a golang module for creating image representation of context data.",
									Opts: TextBlockOpts{TextWrap: true},
								},
							}, 0, 0, 0),
						sampleJpgBlocks[0],
						sampleImageSets[1],
						samplePngBlocks[0],
					}, 0, 0, 0),
			},
		},
		{
			name: "Render Character Avatar with Accessories",
			scene: Scene{
				Main: NewPaneWithShape(
					NewShapeWithObjects([]Column{
						// first column
						{
							Objects: []Tileable{
								&TextBlock{Text: "Charater Sheet", Opts: TextBlockOpts{TextWrap: true}},
								&TextBlock{Text: "An auto-generated image context that embeds every details of the character, including her facial details, hair style, and optional wearings and accessories.", Opts: TextBlockOpts{TextWrap: true}},
							},
						},
						// second column
						{
							Objects: sampleImageSets[0:2],
						},
						// third column
						{
							Objects: sampleImageSets[2:4],
						},
					}), 480, DefaultColPad, DefaultMinPad),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := eng.Render(&tc.scene)
			require.NoError(t, err)
			require.NotNil(t, c.Raw)
			f, err := os.Create("test_output/" + tc.name + ".png")
			require.NoError(t, err)
			defer f.Close()

			err = c.ToPng(f)
			require.NoError(t, err)
		})
	}
}
