package main

import (
	"context"
	"errors"
	"fade"
	"fmt"
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

	o, err := fade.LoadAssets()
	if err != nil {
		log.Fatal(err)
	}

	world := fade.WorldImage(o, fade.WorldMap, fade.WorldMap.Bounds())

	if *addr == "-" {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		profile := colorprofile.Detect(os.Stdout, os.Environ())
		m, opts := fade.NewProgram(ctx, fade.Options{
			Stderr:  os.Stderr,
			Flags:   flag.Args(),
			Profile: profile,
			Assets:  o,
			World:   world,
		})
		p := tea.NewProgram(m, opts...)
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
				return fade.NewProgram(sess.Context(), fade.Options{
					Stderr:  sess.Stderr(),
					Flags:   sess.Command(),
					Profile: colorprofile.ANSI,
					Assets:  o,
					World:   world,
				})
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
