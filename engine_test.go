package imacon

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEngine_Render(t *testing.T) {
	// create test output folder
	_ = os.Mkdir("test_output", os.ModePerm)

	eng := New(Config{
		MaxCanvasWidth:  400,
		MaxCanvasHeight: 320,
	})

	t.Run("Render Simple TextBlock", func(t *testing.T) {
		scene := Scene{
			Main: &Pane{
				Objects: []Tileable{
					&TextBlock{Text: "Hello, World!"},
				},
			},
		}
		c, err := eng.Render(&scene)
		require.NoError(t, err)
		require.NotNil(t, c.Raw)

		// create a file writer
		f, err := os.Create("test_output/render_textblock.png")
		require.NoError(t, err)
		defer f.Close()

		err = c.ToPng(f)
		require.NoError(t, err)
	})

}
