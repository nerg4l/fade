package main

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/anthonynsimon/bild/transform"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type SpriteSheet interface {
	image.Image
	SubImage(image.Rectangle) image.Image
}

func openSpriteSheet(name string) (SpriteSheet, error) {
	f, err := content.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	i, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return i.(SpriteSheet), nil
}

type spriteTrainer struct {
	id      string
	sprites map[string][]image.Image
	face    string
	anim    int

	lock bool
}

type trainerOptions struct {
	FrontIdle image.Image
	FrontWalk image.Image
	BackIdle  image.Image
	BackWalk  image.Image
	SideIdle  image.Image
	SideWalk  image.Image
}

func newTrainer(r *lipgloss.Renderer, o trainerOptions) (*spriteTrainer, error) {
	m := spriteTrainer{id: generateId(), face: "down"}
	m.sprites = map[string][]image.Image{
		"down":  make([]image.Image, 4),
		"up":    make([]image.Image, 4),
		"left":  make([]image.Image, 2),
		"right": make([]image.Image, 2),
	}

	m.sprites["down"][0] = o.FrontIdle
	m.sprites["down"][1] = o.FrontWalk
	m.sprites["down"][2] = m.sprites["down"][0]
	m.sprites["down"][3] = transform.FlipH(o.FrontWalk)

	m.sprites["up"][0] = o.BackIdle
	m.sprites["up"][1] = o.BackWalk
	m.sprites["up"][2] = m.sprites["up"][0]
	m.sprites["up"][3] = transform.FlipH(o.BackWalk)

	m.sprites["left"][0] = o.SideIdle
	m.sprites["left"][1] = o.SideWalk
	m.sprites["right"][0] = transform.FlipH(o.SideIdle)
	m.sprites["right"][1] = transform.FlipH(o.SideWalk)

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

func (m spriteTrainer) View() image.Image {
	return m.sprites[m.face][m.anim]
}

func imageAsString(r *lipgloss.Renderer, img image.Image) string {
	var b strings.Builder
	rec := img.Bounds()
	for y := 0; y < rec.Dy(); y += 2 {
		if y != 0 {
			b.WriteString("\n")
		}
		for x := 0; x < rec.Dx(); x++ {
			top := img.At(rec.Min.X+x, rec.Min.Y+y)
			bottom := img.At(rec.Min.X+x, rec.Min.Y+y+1)
			b.WriteString(r.NewStyle().
				Foreground(colorize(top)).
				Background(colorize(bottom)).
				Render("▀"))
		}
	}
	return b.String()
}

func colorize(c color.Color) lipgloss.TerminalColor {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return lipgloss.CompleteColor{TrueColor: PalletWhite, ANSI256: "250", ANSI: "7"}
	}
	hex := fmt.Sprintf("#%02x%02x%02x", uint8(r), uint8(g), uint8(b))
	switch hex {
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
