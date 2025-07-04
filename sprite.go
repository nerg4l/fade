package main

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"github.com/anthonynsimon/bild/transform"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"image"
	"image/color"
	"image/png"
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

type Sprite[T any] struct {
	Pos   Point
	Model T

	Focused   bool
	TargetPos Point
}

type spriteTrainer struct {
	id      string
	sprites map[string][]image.Image
	face    string
	anim    int
	lock    bool
}

type trainerAssets struct {
	FrontIdle image.Image
	FrontWalk image.Image
	BackIdle  image.Image
	BackWalk  image.Image
	SideIdle  image.Image
	SideWalk  image.Image
}

func newTrainer(a trainerAssets) spriteTrainer {
	m := spriteTrainer{id: generateId(), face: "down"}
	m.sprites = map[string][]image.Image{
		"down":  make([]image.Image, 4),
		"up":    make([]image.Image, 4),
		"left":  make([]image.Image, 2),
		"right": make([]image.Image, 2),
	}

	m.sprites["down"][0] = a.FrontIdle
	m.sprites["down"][1] = a.FrontWalk
	m.sprites["down"][2] = m.sprites["down"][0]
	m.sprites["down"][3] = transform.FlipH(a.FrontWalk)

	m.sprites["up"][0] = a.BackIdle
	m.sprites["up"][1] = a.BackWalk
	m.sprites["up"][2] = m.sprites["up"][0]
	m.sprites["up"][3] = transform.FlipH(a.BackWalk)

	m.sprites["left"][0] = a.SideIdle
	m.sprites["left"][1] = a.SideWalk
	m.sprites["right"][0] = transform.FlipH(a.SideIdle)
	m.sprites["right"][1] = transform.FlipH(a.SideWalk)

	return m
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
	PalletWhite     = color.NRGBA{R: 0xf8, G: 0xf8, B: 0xf8, A: 0xff}
	PalletBlack     = color.NRGBA{R: 0x14, G: 0x14, B: 0x14, A: 0xff}
	PalletHighlight = color.NRGBA{R: 0xa8, G: 0xa8, B: 0xa8, A: 0xff}
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
				m.lock = true
				m.anim++
				cmds = append(cmds, doTick(7*frameDuration, m.id), func() tea.Msg {
					return moveMsg{Direction: k}
				}, func() tea.Msg {
					return soundMsg("walk")
				})
			} else {
				m.face = k
			}
		}
	case tickMsg:
		if m.lock {
			m.lock = false
		}
		if m.id != msg.ID {
			break
		}
		if m.anim%2 == 1 {
			m.anim++
		}
	}
	m.anim %= len(m.sprites[m.face])
	return m, tea.Batch(cmds...)
}

func (m spriteTrainer) View() image.Image {
	return m.sprites[m.face][m.anim]
}

func colorize(c color.Color) lipgloss.TerminalColor {
	switch c {
	case PalletWhite:
		return lipgloss.CompleteColor{TrueColor: "#f8f8f8", ANSI256: "250", ANSI: "7"}
	case PalletBlack:
		return lipgloss.CompleteColor{TrueColor: "#141414", ANSI256: "237", ANSI: "0"}
	case PalletHighlight:
		return lipgloss.CompleteColor{TrueColor: "#cc0000", ANSI256: "124", ANSI: "1"}
	default:
		return lipgloss.Color("3")
	}
}
