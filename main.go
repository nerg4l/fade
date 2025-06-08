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
	flag "github.com/spf13/pflag"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Game struct {
	trainer Sprite[spriteTrainer]
	options gameOptions
	sound   *soundServer

	r *lipgloss.Renderer

	world      *image.NRGBA
	pixelCache map[struct{ Top, Bottom color.Color }]string
}

type gameOptions struct {
	Trainer trainerOptions
	Blank   image.Image
	Grass   image.Image
	Brick   image.Image
}

func newGame(r *lipgloss.Renderer, o gameOptions, world *image.NRGBA) Game {
	trainer := newTrainer(o.Trainer)

	ss := soundServer{w: io.Discard, c: make(chan soundMsg), lc: make(chan soundLoopMsg)}

	return Game{
		trainer: Sprite[spriteTrainer]{
			Pos:   Point{4 * 16, 4 * 16},
			Model: trainer,

			TargetPos: Point{4 * 16, 4 * 16},
			Focused:   true,
		},
		options: o,
		sound:   &ss,
		r:       r,

		world:      world,
		pixelCache: make(map[struct{ Top, Bottom color.Color }]string),
	}
}

func (g Game) WithSound(w io.Writer) Game {
	g.sound.w = w
	return g
}

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

func (g Game) Init() tea.Cmd {
	return tea.Batch(
		tea.HideCursor,
		g.trainer.Model.Init(),
	)
}

var frameDuration = 30 * time.Millisecond

func AnimateInbetween() tea.Cmd {
	return tea.Tick(frameDuration, func(_ time.Time) tea.Msg { return animateMoveMsg{} })
}

func (g Game) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch k := msg.String(); k {
		case "ctrl+c":
			return g, tea.Quit
		}
	case moveMsg:
		bounds := image.Rectangle{
			image.Point{g.world.Rect.Min.X + (2 * 16) + 1, g.world.Rect.Min.Y + (2 * 16) + 1},
			image.Point{g.world.Rect.Max.X - 16 - ((2 * 16) + 1), g.world.Rect.Max.Y - 16 - ((2 * 16) + 1)},
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

var worldMap = func() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, 18, 18))
	img.Pix = []byte{
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'G', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'G', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'G', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'G', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', ' ', 'G', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'G', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'G', 'G', ' ', 'G', 'G', 'G', ' ', 'G', 'G', 'G', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
		'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B', 'B',
	}
	return img
}()

func worldImage(o gameOptions, m *image.Gray) *image.NRGBA {
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

func (g Game) View() string {
	w, h := 5*16, 5*16
	visibleArea := image.NewNRGBA(image.Rect(0, 0, w, h))
	{
		src := g.world.SubImage(image.Rect(
			g.trainer.Pos.X-(w/2-8), g.trainer.Pos.Y-(h/2-8),
			g.trainer.Pos.X+(w/2+8), g.trainer.Pos.Y+(h/2+8),
		))
		dp := image.Point{}
		r := image.Rectangle{dp, dp.Add(src.Bounds().Size())}
		draw.Draw(visibleArea, r, src, src.Bounds().Min, draw.Src)
	}
	{
		src := g.trainer.Model.View()
		dp := image.Point{w/2 - 8, h/2 - 8}
		r := image.Rectangle{dp, dp.Add(src.Bounds().Size())}
		draw.Draw(visibleArea, r, src, src.Bounds().Min, draw.Over)
	}
	return g.imageAsString(visibleArea)
}

func (g Game) imageAsString(img image.Image) string {
	var b strings.Builder
	rec := img.Bounds()

	for y := 0; y < rec.Dy(); y += 2 {
		if y != 0 {
			b.WriteString("\n")
		}
		for x := 0; x < rec.Dx(); x++ {
			top := img.At(rec.Min.X+x, rec.Min.Y+y)
			bottom := img.At(rec.Min.X+x, rec.Min.Y+y+1)
			k := struct{ Top, Bottom color.Color }{Top: top, Bottom: bottom}
			s, ok := g.pixelCache[k]
			if !ok {
				s = g.r.NewStyle().
					Foreground(colorize(top)).
					Background(colorize(bottom)).
					Render("▀")
				g.pixelCache[k] = s
			}
			b.WriteString(s)
		}
	}
	return b.String()
}

func extendGameWithArgs(g Game, sound io.Writer, args []string) Game {
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

	world := worldImage(o, worldMap)

	if *addr == "-" {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		m := newGame(lipgloss.NewRenderer(os.Stdout), o, world)
		m = extendGameWithArgs(m, os.Stderr, flag.Args())
		go m.sound.Start(ctx)
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
		ssh.PublicKeyAuth(func(ssh.Context, ssh.PublicKey) bool { return true }), // Accept any public key.
		ssh.PasswordAuth(func(ssh.Context, string) bool { return false }),        // Do not accept password auth.
		wish.WithMiddleware(
			bubbletea.Middleware(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				m := newGame(bubbletea.MakeRenderer(sess), o, world)
				m = extendGameWithArgs(m, sess.Stderr(), sess.Command())
				go m.sound.Start(sess.Context())
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
