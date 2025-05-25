package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/disintegration/imaging"
	flag "github.com/spf13/pflag"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

//go:embed sprite/*
var content embed.FS

type model struct {
	sprites []string
	anim    int
}

func newModel() (*model, error) {
	m := model{}
	m.sprites = make([]string, 4)

	idle, err := openImage("sprite/front_idle.png")
	if err != nil {
		return nil, err
	}
	m.sprites[0] = imageAsString(idle)
	m.sprites[2] = m.sprites[0]
	walk, err := openImage("sprite/front_walk.png")
	if err != nil {
		return nil, err
	}
	m.sprites[1] = imageAsString(walk)
	m.sprites[3] = imageAsString(imaging.FlipH(walk))

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

func imageAsString(dec image.Image) string {
	var b strings.Builder
	rec := dec.Bounds()
	for y := 0; y < rec.Dy(); y += 2 {
		for x := 0; x < rec.Dx(); x++ {
			top := colorize(dec.At(x, y))
			bottom := colorize(dec.At(x, y+1))
			if Luminance(top) > Luminance(bottom) {
				b.WriteString(lipgloss.NewStyle().
					Foreground(top).
					Background(bottom).
					Render("▀"))
			} else if Luminance(top) < Luminance(bottom) {
				b.WriteString(lipgloss.NewStyle().
					Foreground(bottom).
					Background(top).
					Render("▄"))
			} else if Luminance(top) < Luminance(color.RGBA{R: 128, G: 128, B: 128, A: 255}) {
				b.WriteString(lipgloss.NewStyle().
					Foreground(top).
					Background(lipgloss.Color("#ffffff")).
					Render("█"))
			} else {
				b.WriteString(lipgloss.NewStyle().
					Background(bottom).
					Render(" "))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func Luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.2126*toSRGB(uint8(r)) + 0.7152*toSRGB(uint8(g)) + 0.0722*toSRGB(uint8(b))
}

func toSRGB(i uint8) float64 {
	v := float64(i) / 255
	if v <= 0.04045 {
		return v / 12.92
	} else {
		return math.Pow((v+0.055)/1.055, 2.4)
	}
}

func (m model) Init() tea.Cmd {
	return tea.HideCursor
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "down":
			m.anim++
			m.anim %= len(m.sprites)
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() string {
	return m.sprites[m.anim]
}

func colorize(color color.Color) lipgloss.Color {
	r, g, b, a := color.RGBA()
	if a == 0 {
		return lipgloss.Color("#ffffff")
	}
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func main() {
	addr := flag.StringP("addr", "a", "0.0.0.0:5000", "SSH server port")
	flag.Parse()

	options := []tea.ProgramOption{
		tea.WithAltScreen(),
	}

	m, err := newModel()
	if err != nil {
		log.Fatal(err)
	}

	if *addr == "-" {
		p := tea.NewProgram(m, options...)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
		return
	}

	s, err := wish.NewServer(
		wish.WithAddress(*addr),
		wish.WithHostKeyPath("storage/.ssh/id_fade"),
		// Accept any public key.
		ssh.PublicKeyAuth(func(ssh.Context, ssh.PublicKey) bool { return true }),
		// Do not accept password auth.
		ssh.PasswordAuth(func(ssh.Context, string) bool { return false }),
		wish.WithMiddleware(
			bubbletea.Middleware(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				return m, append(options, tea.WithContext(sess.Context()))
			}),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Println("Starting SSH server", "addr", *addr)
	go func(cancel context.CancelFunc) {
		defer cancel()
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "Error starting SSH server:", err)
		}
	}(cancel)

	<-ctx.Done()
	log.Println("Stopping SSH server")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, "Error stopping SSH server:", err)
	}
}
