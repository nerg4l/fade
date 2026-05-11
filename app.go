package fade

import (
	"context"
	"image"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
)

type Options struct {
	Profile colorprofile.Profile
	Stderr  io.Writer
	Assets  GameAssets
	World   *image.NRGBA
	Flags   []string
}

func NewProgram(ctx context.Context, o Options) (tea.Model, []tea.ProgramOption) {
	m := newGameSession(o.Profile, o.Assets, o.World)
	m = extendGameWithArgs(m, o.Stderr, o.Flags)
	go m.sound.Start(ctx)
	return m, []tea.ProgramOption{tea.WithContext(ctx), tea.WithFPS(25)}
}

// A Point is an X, Y coordinate pair. The axes increase right and down.
type Point struct {
	X, Y int
}

func LoadAssets() (GameAssets, error) {
	var a GameAssets
	var err error
	tile, err := openSpriteSheet("sprite/tile.png")
	if err != nil {
		return a, err
	}
	a.Blank = tile.SubImage(image.Rect(0, 0, 8, 8))
	a.Grass = tile.SubImage(image.Rect(8, 0, 16, 8))
	a.Brick = tile.SubImage(image.Rect(0, 8, 8, 16))
	a.Bush = tile.SubImage(image.Rect(8, 8, 16, 16))
	trainer, err := openSpriteSheet("sprite/trainer.png")
	if err != nil {
		return a, err
	}
	for y := 0; y < (trainer.Bounds().Dy() / 16); y++ {
		for x := 0; x < (trainer.Bounds().Dx() / 16); x++ {
			img := trainer.SubImage(image.Rect(x*16, y*16, (x+1)*16, (y+1)*16))
			v := Point{X: x, Y: y}
			switch v {
			case Point{Y: 0, X: 0}:
				a.Trainer.FrontIdle = img
			case Point{Y: 0, X: 1}:
				a.Trainer.FrontWalk = img
			case Point{Y: 1, X: 0}:
				a.Trainer.SideIdle = img
			case Point{Y: 1, X: 1}:
				a.Trainer.SideWalk = img
			case Point{Y: 2, X: 0}:
				a.Trainer.BackIdle = img
			case Point{Y: 2, X: 1}:
				a.Trainer.BackWalk = img
			}
		}
	}
	return a, nil
}
