package fade

import (
	"image"
	"image/draw"
	"time"

	tea "charm.land/bubbletea/v2"
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

var WorldMap = func() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, 14, 14))
	img.Pix = []byte{
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'G', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'B', 'B',
		'B', 'B', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', 'G', ' ', 'b', 'b', 'G', ' ', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', ' ', 'G', 'b', 'b', ' ', 'G', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'B', 'B',
		'B', 'B', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'B', 'B',
		'B', 'B', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'G', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
	}
	return img
}()

func WorldImage(o GameAssets, m *image.Gray, r image.Rectangle) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(r.Bounds().Min.X*8, r.Bounds().Min.Y*8, r.Bounds().Max.X*8, r.Bounds().Max.Y*8))
	for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
		for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
			src := o.Blank
			switch m.GrayAt(x, y).Y {
			case 'B':
				src = o.Brick
			case 'b':
				src = o.Bush
			case 'G':
				src = o.Grass
			}
			dp := image.Point{(x - m.Bounds().Min.X) * 8, (y - m.Bounds().Min.Y) * 8}
			r := image.Rectangle{dp, dp.Add(src.Bounds().Size())}
			draw.Draw(img, r, src, src.Bounds().Min, draw.Src)
		}
	}
	return img
}

type World struct {
	m       *image.Gray
	o       GameAssets
	players map[image.Point][]image.Image
	npcs    map[image.Point][]image.Image
}

func NewWorld(o GameAssets) *World {
	return &World{
		m:       WorldMap,
		o:       o,
		players: make(map[image.Point][]image.Image),
		npcs:    make(map[image.Point][]image.Image),
	}
}

func (w *World) SumImage(r image.Rectangle) image.Image {
	img := image.NewNRGBA(image.Rect(r.Min.X*16, r.Min.Y*16, r.Max.X*16, r.Max.Y*16))
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 8 {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x += 8 {
			src := w.o.Blank
			switch w.m.GrayAt(x/8, y/8).Y {
			case 'B':
				src = w.o.Brick
			case 'b':
				src = w.o.Bush
			case 'G':
				src = w.o.Grass
			}
			dp := image.Point{x, y}
			r := image.Rectangle{dp, dp.Add(src.Bounds().Size())}
			draw.Draw(img, r, src, src.Bounds().Min, draw.Src)
		}
	}
	for x := 0; x < r.Min.X; x++ {
		for y := 0; y < r.Min.Y; y++ {
			p, ok := w.players[image.Point{x, y}]
			if !ok {
				continue
			}
			for _, v := range p {
				draw.Draw(img, v.Bounds(), v, image.Point{}, draw.Over)
			}

			n, ok := w.npcs[image.Point{x, y}]
			if !ok {
				continue
			}
			for _, v := range n {
				draw.Draw(img, v.Bounds(), v, image.Point{}, draw.Over)
			}
		}
	}
	return img
}
