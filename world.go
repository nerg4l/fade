package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"image"
	"image/draw"
	"time"
)

var frameDuration = 30 * time.Millisecond

type tickMsg struct {
	ID string
	T  time.Time
}

func doTick(d time.Duration, id string) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg{ID: id, T: t}
	})
}

type moveMsg struct {
	Direction string
}

type animateMoveMsg struct{}

func AnimateInbetween() tea.Cmd {
	return tea.Tick(frameDuration, func(_ time.Time) tea.Msg { return animateMoveMsg{} })
}

var worldMap = func() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, 14, 14))
	img.Pix = []byte{
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'G', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'B', 'B',
		'B', 'B', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'B', 'B',
		'B', 'B', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'G', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
	}
	return img
}()

func worldImage(o gameAssets, m *image.Gray) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, m.Bounds().Dx()*8, m.Bounds().Dy()*8))
	for y := 0; y < m.Bounds().Dy(); y++ {
		for x := 0; x < m.Bounds().Dx(); x++ {
			src := o.Blank
			switch m.GrayAt(x, y).Y {
			case 'B':
				src = o.Brick
			case 'G':
				src = o.Grass
			}
			dp := image.Point{x * 8, y * 8}
			r := image.Rectangle{dp, dp.Add(src.Bounds().Size())}
			draw.Draw(img, r, src, src.Bounds().Min, draw.Src)
		}
	}
	return img
}
