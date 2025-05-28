package main

import (
	"context"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
	flag "github.com/spf13/pflag"
	"image"
	"image/draw"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Game struct {
	trainer spriteTrainer
	options gameOptions

	r *lipgloss.Renderer

	trainerPosition Point
}

type gameOptions struct {
	Trainer trainerOptions
	Blank   image.Image
	Grass   image.Image
	Brick   image.Image
}

func newGame(r *lipgloss.Renderer, o gameOptions) *Game {
	trainer, err := newTrainer(r, o.Trainer)
	if err != nil {
		return nil
	}

	return &Game{
		trainer: *trainer,
		options: o,
		r:       r,

		trainerPosition: Point{4, 4},
	}
}

type tickMsg struct {
	ID string
	T  time.Time
}

func doTick(id string) tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{ID: id, T: t}
	})
}

type moveMsg struct {
	Direction string
}

func (g *Game) Init() tea.Cmd {
	return tea.Batch(
		tea.HideCursor,
		doTick("root"),
		g.trainer.Init(),
	)
}

func (g *Game) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch k := msg.String(); k {
		case "ctrl+c":
			return g, tea.Quit
		}
	case tickMsg:
		if msg.ID != "root" {
			break
		}
		cmds = append(cmds, doTick("root"))
	case moveMsg:
		switch msg.Direction {
		case "up":
			if g.trainerPosition.Y > 2 {
				g.trainerPosition.Y--
			}
		case "down":
			if g.trainerPosition.Y < 6 {
				g.trainerPosition.Y++
			}
		case "left":
			if g.trainerPosition.X > 2 {
				g.trainerPosition.X--
			}
		case "right":
			if g.trainerPosition.X < 6 {
				g.trainerPosition.X++
			}
		}
	}
	g.trainer, cmd = g.trainer.Update(msg)
	cmds = append(cmds, cmd)

	return g, tea.Batch(cmds...)
}

var worldMap = []byte{
	'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
	'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
	'B', 'B', 'G', ' ', 'G', ' ', 'G', ' ', 'B', 'B',
	'B', 'B', ' ', 'G', ' ', 'G', ' ', 'G', 'B', 'B',
	'B', 'B', 'G', ' ', 'G', 'G', 'G', ' ', 'B', 'B',
	'B', 'B', ' ', 'G', 'G', 'G', ' ', 'G', 'B', 'B',
	'B', 'B', 'G', ' ', 'G', ' ', 'G', ' ', 'B', 'B',
	'B', 'B', ' ', 'G', ' ', 'G', ' ', 'G', 'B', 'B',
	'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
	'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
}

func (g *Game) View() string {
	visibleArea := image.NewRGBA(image.Rect(0, 0, 3*16, 3*16))
	center := image.Point{g.trainerPosition.X * 8, g.trainerPosition.Y * 8}
	for y := 0; y < visibleArea.Rect.Dy(); y += 8 {
		for x := 0; x < visibleArea.Rect.Dx(); x += 8 {
			dp := image.Point{x, y}
			i, j := (center.X-16+x)/8, (center.Y-16+y)/8
			src := g.options.Blank
			switch worldMap[(j*10)+i] {
			case 'B':
				src = g.options.Brick
			case 'G':
				src = g.options.Grass
			}
			r := image.Rectangle{dp, dp.Add(src.Bounds().Size())}
			draw.Draw(visibleArea, r, src, src.Bounds().Min, draw.Src)
		}
	}
	src := g.trainer.View()
	dp := image.Point{16, 16}
	r := image.Rectangle{dp, dp.Add(src.Bounds().Size())}
	draw.Draw(visibleArea, r, src, src.Bounds().Min, draw.Over)
	return imageAsString(g.r, visibleArea)
}

func main() {
	addr := flag.StringP("addr", "a", "0.0.0.0:5000", "SSH server port")
	flag.Parse()

	options := []tea.ProgramOption{
		tea.WithAltScreen(),
	}

	o, err := loadOptions()
	if err != nil {
		log.Fatal(err)
	}

	if *addr == "-" {
		m := newGame(lipgloss.NewRenderer(os.Stdout), o)
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
			bubbletea.MiddlewareWithColorProfile(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				m := newGame(bubbletea.MakeRenderer(sess), o)
				return m, append(options, tea.WithContext(sess.Context()))
			}, termenv.ANSI),
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

// A Point is an X, Y coordinate pair. The axes increase right and down.
type Point struct {
	X, Y int
}

func loadOptions() (gameOptions, error) {
	var o gameOptions
	var err error
	tile, err := openSpriteSheet("sprite/tile.png")
	if err != nil {
		return o, err
	}
	o.Blank = tile.SubImage(image.Rect(0, 0, 8, 8))
	o.Grass = tile.SubImage(image.Rect(8, 0, 16, 8))
	o.Brick = tile.SubImage(image.Rect(0, 8, 8, 16))
	trainer, err := openSpriteSheet("sprite/trainer.png")
	if err != nil {
		return o, err
	}
	for y := 0; y < (trainer.Bounds().Dy() / 16); y++ {
		for x := 0; x < (trainer.Bounds().Dx() / 16); x++ {
			img := trainer.SubImage(image.Rect(x*16, y*16, (x+1)*16, (y+1)*16))
			v := Point{X: x, Y: y}
			switch v {
			case Point{Y: 0, X: 0}:
				o.Trainer.FrontIdle = img
			case Point{Y: 0, X: 1}:
				o.Trainer.FrontWalk = img
			case Point{Y: 1, X: 0}:
				o.Trainer.SideIdle = img
			case Point{Y: 1, X: 1}:
				o.Trainer.SideWalk = img
			case Point{Y: 2, X: 0}:
				o.Trainer.BackIdle = img
			case Point{Y: 2, X: 1}:
				o.Trainer.BackWalk = img
			}
		}
	}
	return o, nil
}
