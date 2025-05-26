package main

import (
	"context"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
	flag "github.com/spf13/pflag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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
			bubbletea.MiddlewareWithColorProfile(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				m.r = bubbletea.MakeRenderer(sess)
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
