package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/ssh"
	flag "github.com/spf13/pflag"
)

func main() {
	addr := flag.StringP("addr", "a", "0.0.0.0:5000", "SSH server port")
	flag.Parse()

	o, err := loadAssets()
	if err != nil {
		log.Fatal(err)
	}

	world := worldImage(o, worldMap)

	if *addr == "-" {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		m := newGameSession(colorprofile.Detect(os.Stdout, os.Environ()), o, world)
		m = extendGameWithArgs(m, os.Stderr, flag.Args())
		go m.sound.Start(ctx)
		p := tea.NewProgram(m, tea.WithFPS(25))
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
				m := newGameSession(colorprofile.ANSI, o, world)
				m = extendGameWithArgs(m, sess.Stderr(), sess.Command())
				go m.sound.Start(sess.Context())
				return m, []tea.ProgramOption{tea.WithContext(sess.Context()), tea.WithFPS(25)}
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

func loadAssets() (gameAssets, error) {
	var a gameAssets
	var err error
	tile, err := openSpriteSheet("sprite/tile.png")
	if err != nil {
		return a, err
	}
	a.Blank = tile.SubImage(image.Rect(0, 0, 8, 8))
	a.Grass = tile.SubImage(image.Rect(8, 0, 16, 8))
	a.Brick = tile.SubImage(image.Rect(0, 8, 8, 16))
	trainer, err := openSpriteSheet("sprite/trainer.png")
	if err != nil {
		return a, err
	}
	for y := 0; y < (trainer.Bounds().Dy() / 16); y++ {
		for x := 0; x < (trainer.Bounds().Dx() / 16); x++ {
			img := trainer.SubImage(image.Rect(x*16, y*16, (x+1)*16, (y+1)*16))
			v := Point{X: x, Y: y}
			switch v {
			case Point{Y: 0, X: 0}:
				a.Trainer.FrontIdle = img
			case Point{Y: 0, X: 1}:
				a.Trainer.FrontWalk = img
			case Point{Y: 1, X: 0}:
				a.Trainer.SideIdle = img
			case Point{Y: 1, X: 1}:
				a.Trainer.SideWalk = img
			case Point{Y: 2, X: 0}:
				a.Trainer.BackIdle = img
			case Point{Y: 2, X: 1}:
				a.Trainer.BackWalk = img
			}
		}
	}
	return a, nil
}
