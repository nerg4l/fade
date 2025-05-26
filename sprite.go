package main

import (
	"embed"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	"image/png"
	"strings"
)

//go:embed sprite/*
var content embed.FS

type spriteTrainer struct {
	sprites map[string][]image.Image
	face    string
	anim    int

	r *lipgloss.Renderer
}

func newModel() (*spriteTrainer, error) {
	m := spriteTrainer{face: "down"}
	m.r = lipgloss.DefaultRenderer()
	m.sprites = map[string][]image.Image{
		"down":  make([]image.Image, 4),
		"up":    make([]image.Image, 4),
		"left":  make([]image.Image, 2),
		"right": make([]image.Image, 2),
	}

	idle, err := openImage("sprite/front_idle.png")
	if err != nil {
		return nil, err
	}
	walk, err := openImage("sprite/front_walk.png")
	if err != nil {
		return nil, err
	}
	m.sprites["down"][0] = idle
	m.sprites["down"][1] = walk
	m.sprites["down"][2] = m.sprites["down"][0]
	m.sprites["down"][3] = imaging.FlipH(walk)

	idle, err = openImage("sprite/back_idle.png")
	if err != nil {
		return nil, err
	}
	walk, err = openImage("sprite/back_walk.png")
	if err != nil {
		return nil, err
	}
	m.sprites["up"][0] = idle
	m.sprites["up"][1] = walk
	m.sprites["up"][2] = m.sprites["up"][0]
	m.sprites["up"][3] = imaging.FlipH(walk)

	idle, err = openImage("sprite/side_idle.png")
	if err != nil {
		return nil, err
	}
	walk, err = openImage("sprite/side_walk.png")
	if err != nil {
		return nil, err
	}
	m.sprites["left"][0] = idle
	m.sprites["left"][1] = walk
	m.sprites["right"][0] = imaging.FlipH(idle)
	m.sprites["right"][1] = imaging.FlipH(walk)

	return &m, nil
}

func openImage(name string) (image.Image, error) {
	f, err := content.Open(name)
	if err != nil {
		return nil, err
	}
	i, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return i, nil
}

var (
	PalletWhite     = "#f8f8f8"
	PalletBlack     = "#141414"
	PalletHighlight = "#a8a8a8"
)

func imageAsString(r *lipgloss.Renderer, dec image.Image) string {
	var b strings.Builder
	rec := dec.Bounds()
	for y := 0; y < rec.Dy(); y += 2 {
		for x := 0; x < rec.Dx(); x++ {
			top := dec.At(x, y)
			bottom := dec.At(x, y+1)
			b.WriteString(r.NewStyle().
				Foreground(colorize(top)).
				Background(colorize(bottom)).
				Render("▀"))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m spriteTrainer) Init() tea.Cmd {
	return tea.HideCursor
}

func (m spriteTrainer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "down":
			m.face = "down"
			m.anim++
		case "up":
			m.face = "up"
			m.anim++
		case "left":
			m.face = "left"
			m.anim++
		case "right":
			m.face = "right"
			m.anim++
		}
	}
	m.anim %= len(m.sprites[m.face])
	return m, nil
}

func (m spriteTrainer) View() string {
	return imageAsString(m.r, m.sprites[m.face][m.anim])
}

func colorize(color color.Color) lipgloss.TerminalColor {
	r, g, b, a := color.RGBA()
	if a == 0 {
		return lipgloss.CompleteColor{TrueColor: PalletWhite, ANSI256: "250", ANSI: "7"}
	}
	c := fmt.Sprintf("#%02x%02x%02x", uint8(r), uint8(g), uint8(b))
	switch c {
	case PalletWhite:
		return lipgloss.CompleteColor{TrueColor: PalletWhite, ANSI256: "250", ANSI: "7"}
	case PalletBlack:
		return lipgloss.CompleteColor{TrueColor: PalletBlack, ANSI256: "237", ANSI: "0"}
	case PalletHighlight:
		return lipgloss.CompleteColor{TrueColor: "#cc0000", ANSI256: "124", ANSI: "1"}
	default:
		return lipgloss.Color("3")
	}
}
