package main

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	"image/png"
	"strings"
)

var (
	TileMaxPoint = image.Point{X: 16, Y: 16}
)

//go:embed sprite/*
var content embed.FS

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

type spriteTrainer struct {
	id      string
	sprites map[string][]image.Image
	face    string
	anim    int

	lock bool

	background image.Image

	r *lipgloss.Renderer
}

func newTrainer() (*spriteTrainer, error) {
	m := spriteTrainer{id: generateId(), face: "down"}
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

func generateId() string {
	b := make([]byte, 8)
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	id := base64.RawStdEncoding.EncodeToString(b)
	return id
}

var (
	PalletWhite     = "#f8f8f8"
	PalletBlack     = "#141414"
	PalletHighlight = "#a8a8a8"
)

func (m spriteTrainer) Init() tea.Cmd {
	return nil
}

func (m spriteTrainer) WithBackground(i image.Image) spriteTrainer {
	m.background = i
	return m
}

func (m spriteTrainer) Update(msg tea.Msg) (spriteTrainer, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.lock {
			break
		}
		switch k := msg.String(); k {
		case "down", "up", "left", "right":
			if m.face == k {
				m.anim++
			} else {
				m.face = k
				m.anim = 1
			}
			cmds = append(cmds, doTick(m.id))
			m.lock = true
		}
	case tickMsg:
		if m.id != msg.ID {
			break
		}
		if m.anim%2 == 1 {
			m.anim++
			m.lock = false
		}
	}
	m.anim %= len(m.sprites[m.face])
	return m, tea.Batch(cmds...)
}

func (m spriteTrainer) View() string {
	if m.r == nil {
		return ""
	}
	return imageAsString(m.r, m.sprites[m.face][m.anim], m.background)
}

func imageAsString(r *lipgloss.Renderer, layers ...image.Image) string {
	var b strings.Builder
	var img image.Image = image.NewRGBA(image.Rect(0, 0, TileMaxPoint.X, TileMaxPoint.Y))
	rec := img.Bounds()
	for i := 0; i < len(layers); i++ {
		if layers[i] == nil {
			continue
		}
		img = imaging.Overlay(layers[i], img, image.Point{}, 1)
	}
	for y := 0; y < rec.Dy(); y += 2 {
		if y != 0 {
			b.WriteString("\n")
		}
		for x := 0; x < rec.Dx(); x++ {
			top := img.At(x, y)
			bottom := img.At(x, y+1)
			b.WriteString(r.NewStyle().
				Foreground(colorize(top)).
				Background(colorize(bottom)).
				Render("▀"))
		}
	}
	return b.String()
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
