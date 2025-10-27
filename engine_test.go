package imacon

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEngine_Render(t *testing.T) {
	// create test output folder
	_ = os.Mkdir("test_output", os.ModePerm)

	eng := New(Config{
		MaxCanvasWidth:  2048,
		MaxCanvasHeight: 2048,
	})

	sampleJpgBlocks := make([]Tileable, 10)
	for i := range sampleJpgBlocks {
		sampleJpgFile, err := os.Open("assets/samples/sample_1.jpg")
		require.NoError(t, err)
		defer sampleJpgFile.Close()
		sampleJpgBlocks[i] = NewImageBlock(sampleJpgFile, fmt.Sprintf("Sample Avatar %d", i+1))
	}

	var sampleImageSets []Tileable

	sampleFile, err := os.Open("assets/samples/sample_1.jpg")
	require.NoError(t, err)
	defer sampleFile.Close()
	sampleImageSets = append(sampleImageSets, NewImageBlock(sampleFile, "Avatar"))

	sampleFile, err = os.Open("assets/samples/sample_2.jpg")
	require.NoError(t, err)
	defer sampleFile.Close()
	sampleImageSets = append(sampleImageSets, NewImageBlock(sampleFile, "T-shirt"))

	sampleFile, err = os.Open("assets/samples/sample_3.jpg")
	require.NoError(t, err)
	defer sampleFile.Close()
	sampleImageSets = append(sampleImageSets, NewImageBlock(sampleFile, "Necklace"))

	sampleFile, err = os.Open("assets/samples/sample_4.jpg")
	require.NoError(t, err)
	defer sampleFile.Close()
	sampleImageSets = append(sampleImageSets, NewImageBlock(sampleFile, "Hairstyle"))

	testCases := []struct {
		name  string
		scene Scene
	}{
		{
			name: "Render Simple TextBlock",
			scene: Scene{
				Main: &Pane{
					Objects: []Tileable{
						&TextBlock{Text: "Hello, World!"},
					},
				},
			},
		},
		{
			name: "Render Paragraph",
			scene: Scene{
				Main: &Pane{
					Objects: []Tileable{
						&TextBlock{Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", Opts: TextBlockOpts{TextWrap: true}},
					},
				},
			},
		},
		{
			name: "Render multiple TextBlocks",
			scene: Scene{
				Main: &Pane{
					Objects: []Tileable{
						&TextBlock{Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", Opts: TextBlockOpts{TextWrap: true}},
						&TextBlock{Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", Opts: TextBlockOpts{TextWrap: true}},
						&TextBlock{Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", Opts: TextBlockOpts{TextWrap: true}},
					},
				},
			},
		},
		{
			name: "Render Simple ImageBlock",
			scene: Scene{
				Main: &Pane{
					Objects: sampleJpgBlocks[0:1],
				},
			},
		},
		{
			name: "Render Two ImageBlocks",
			scene: Scene{
				Main: &Pane{
					Objects: sampleJpgBlocks[0:2],
				},
			},
		},
		{
			name: "Render Many ImageBlocks",
			scene: Scene{
				Main: &Pane{
					Objects: sampleJpgBlocks[0:8],
				},
			},
		},
		{
			name: "Render multiple panes",
			scene: Scene{
				Main: &Pane{
					Objects: []Tileable{
						&Pane{
							Objects: sampleJpgBlocks[0:2],
						},
						&Pane{
							Objects: []Tileable{
								&TextBlock{Text: "Below are the accessories:", Opts: TextBlockOpts{TextWrap: true}},
							},
						},
						&Pane{
							Objects: sampleJpgBlocks[2:5],
						},
					},
				},
			},
		},
		{
			name: "Render Character Avatar with Accessories",
			scene: Scene{
				Main: &Pane{
					Objects: sampleImageSets,
				},
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
