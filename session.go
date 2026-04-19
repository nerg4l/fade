package main

import (
	"image"
	"image/color"
	"image/draw"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

type GameSession struct {
	trainer Sprite[spriteTrainer]
	assets  gameAssets
	sound   *soundServer

	p colorprofile.Profile

	world      *image.NRGBA
	pixelCache map[Column]string
}

type gameAssets struct {
	Trainer trainerAssets
	Blank   image.Image
	Grass   image.Image
	Brick   image.Image
}

func newGameSession(p colorprofile.Profile, a gameAssets, world *image.NRGBA) GameSession {
	trainer := newTrainer(a.Trainer)

	ss := soundServer{w: io.Discard, c: make(chan soundMsg), lc: make(chan soundLoopMsg)}

	return GameSession{
		trainer: Sprite[spriteTrainer]{
			Pos:   Point{world.Rect.Dx()/2 - 8, world.Rect.Dy()/2 - 8},
			Model: trainer,

			TargetPos: Point{world.Rect.Dx()/2 - 8, world.Rect.Dy()/2 - 8},
			Focused:   true,
		},
		assets: a,
		sound:  &ss,
		p:      p,

		world:      world,
		pixelCache: make(map[Column]string),
	}
}

func extendGameWithArgs(g GameSession, sound io.Writer, args []string) GameSession {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "audio", "sound":
			if len(args) > i+1 && args[i+1] == "on" {
				g = g.WithSound(sound)
			}
			i++
		}
	}
	return g
}

func (g GameSession) WithSound(w io.Writer) GameSession {
	g.sound.w = w
	return g
}

func (g GameSession) Init() tea.Cmd {
	return tea.Batch(
		g.trainer.Model.Init(),
	)
}

func (g GameSession) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch k := msg.String(); k {
		case "ctrl+c":
			return g, tea.Quit
		}
	case tea.ColorProfileMsg:
		if g.p != msg.Profile {
			g.pixelCache = make(map[Column]string)
		}
		g.p = msg.Profile
		return g, nil
	case moveMsg:
		bounds := image.Rectangle{
			image.Point{g.world.Rect.Min.X + (1 * 16) + 1, g.world.Rect.Min.Y + (1 * 16) + 1},
			image.Point{g.world.Rect.Max.X - 16 - ((1 * 16) + 1), g.world.Rect.Max.Y - 16 - ((1 * 16) + 1)},
		}
		switch msg.Direction {
		case "up":
			if g.trainer.Pos.Y > bounds.Min.Y {
				g.trainer.TargetPos = Point{g.trainer.Pos.X, g.trainer.Pos.Y - 16}
				g.trainer.Focused = false
				cmds = append(cmds, AnimateInbetween())
			}
		case "down":
			if g.trainer.Pos.Y < bounds.Max.Y {
				g.trainer.TargetPos = Point{g.trainer.Pos.X, g.trainer.Pos.Y + 16}
				g.trainer.Focused = false
				cmds = append(cmds, AnimateInbetween())
			}
		case "left":
			if g.trainer.Pos.X > bounds.Min.X {
				g.trainer.TargetPos = Point{g.trainer.Pos.X - 16, g.trainer.Pos.Y}
				g.trainer.Focused = false
				cmds = append(cmds, AnimateInbetween())
			}
		case "right":
			if g.trainer.Pos.X < bounds.Max.X {
				g.trainer.TargetPos = Point{g.trainer.Pos.X + 16, g.trainer.Pos.Y}
				g.trainer.Focused = false
				cmds = append(cmds, AnimateInbetween())
			}
		}
	case animateMoveMsg:
		g.trainer.Pos.X += 2 * Sign(g.trainer.TargetPos.X-g.trainer.Pos.X)
		g.trainer.Pos.Y += 2 * Sign(g.trainer.TargetPos.Y-g.trainer.Pos.Y)
		if g.trainer.Pos == g.trainer.TargetPos {
			g.trainer.Focused = true
		} else {
			cmds = append(cmds, AnimateInbetween())
		}
	}
	if _, ok := msg.(tea.KeyMsg); !ok || g.trainer.Focused {
		g.trainer.Model, cmd = g.trainer.Model.Update(msg)
	}
	g.sound.Update(msg)
	cmds = append(cmds, cmd)

	return g, tea.Batch(cmds...)
}

func Sign(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	}
	return 0
}

func (g GameSession) View() tea.View {
	w, h := 5*16, 5*16
	viewPort := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(viewPort, viewPort.Bounds(), &image.Uniform{PalletBlack}, image.Point{}, draw.Src)
	{
		src := g.world
		r := image.Rectangle{Max: image.Point{X: w, Y: h}}
		draw.Draw(viewPort, r, src, image.Point{
			X: g.trainer.Pos.X - (w/2 - 8), Y: g.trainer.Pos.Y - (h/2 - 8),
		}, draw.Src)
	}
	{
		src := g.trainer.Model.View()
		dp := image.Point{w/2 - 8, h/2 - 8}
		r := image.Rectangle{dp, dp.Add(src.Bounds().Size())}
		draw.Draw(viewPort, r, src, src.Bounds().Min, draw.Over)
	}
	var v tea.View
	v.SetContent(g.imageAsString(viewPort))
	v.Cursor = nil
	v.AltScreen = true
	return v
}

func (g GameSession) imageAsString(img image.Image) string {
	var b strings.Builder
	rec := img.Bounds()
	complete := lipgloss.Complete(g.p)

	for y := rec.Min.Y; y < rec.Max.Y; y += 2 {
		if y != rec.Min.Y {
			b.WriteString("\n")
		}
		for x := rec.Min.X; x < rec.Max.X; x++ {
			top := img.At(x, y)
			bottom := img.At(x, y+1)

			k := Column{Top: top, Bottom: bottom}
			s, ok := g.pixelCache[k]
			if !ok {
				s = lipgloss.NewStyle().
					Foreground(complete(colorize(top))).
					Background(complete(colorize(bottom))).
					Render("▀")
				g.pixelCache[k] = s
			}
			b.WriteString(s)
		}
	}
	return b.String()
}

type Column struct {
	Top, Bottom color.Color
}
