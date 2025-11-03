# Imacon

![Image Context Output](assets/samples/sample_output.png)

Imacon is a golang module for creating image representation of context data.

This module is built to solve the problem of dynamically generating images that represent various types of context data (text, images) in a image representation format that multimodel AI can use it as context input. 

## Features
- Render text blocks with word wrapping.
- Load and display JPEG and PNG images.
- Auto-tiling of multiple objects to fit within a specified canvas size.
- Support for nested panes to create complex layouts.
- Configurable canvas size and font settings.

## Installation

```bash
go get github.com/dannykok/imacon
```

## Requirements

- Go 1.24.1 or higher
- A TrueType font file (e.g., JetBrainsMono-Regular.ttf) placed in `assets/fonts/`

## Usage

### Basic Example

```go
package main

import (
    "os"
    "github.com/dannykok/imacon"
)

func main() {
    // Create engine with configuration
    eng := imacon.New(imacon.Config{
        MaxCanvasWidth:  800,
        MaxCanvasHeight: 600,
        FontSize:        16, // optional
    })

    // Scene is the root struct that holds the main Pane struct
    scene := imacon.Scene{
        Main: &imacon.Pane{
            Objects: []imacon.Tileable{
                &imacon.TextBlock{
                    Text: "Imacon: Image Representation of Context Data",
                    Opts: imacon.TextBlockOpts{TextWrap: true},
                },
            },
        },
    }

    // Render the scene
    canvas, err := eng.Render(&scene)
    if err != nil {
        panic(err)
    }

    // Save to file
    f, _ := os.Create("output.png")
    defer f.Close()
    canvas.ToPng(f)
}
```

### Working with Images

```go
// Load an image
file, _ := os.Open("path/to/image.jpg")
defer file.Close()
imgBlock := imacon.NewImageBlock(file, "My Image")

// Create scene with image
scene := imacon.Scene{
    Main: &imacon.Pane{
        Objects: []imacon.Tileable{imgBlock},
    },
}
```

### Auto-tiling Multiple Objects

```go
// Imacon automatically tiles multiple objects to fit the canvas
scene := imacon.Scene{
    Main: &imacon.Pane{
        Objects: []imacon.Tileable{
            imgBlock1,
            imgBlock2,
            &imacon.TextBlock{Text: "Caption"},
            imgBlock3,
        },
    },
}
```

### Nested Panes

```go
scene := imacon.Scene{
    Main: &imacon.Pane{
        Objects: []imacon.Tileable{
            &imacon.Pane{
                Objects: []imacon.Tileable{img1, img2},
            },
            &imacon.TextBlock{Text: "Section Title"},
            &imacon.Pane{
                Objects: []imacon.Tileable{img3, img4, img5},
            },
        },
    },
}
```

## Testing

```bash
go test -v ./...
```

Test outputs will be saved to `test_output/` directory.

## Dependencies
- [fogleman/gg](https://github.com/fogleman/gg) - for canvas drawing.
