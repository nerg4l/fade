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
	flag "github.com/spf13/pflag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type model struct{}

func (m model) Init() tea.Cmd {
	return tea.HideCursor
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return `
   ▄▀▀▀▀▀▀▄
  █        █
 ██▀▄▄▄▄▄▄▀██
█ ▀  ▄  ▄  ▀ █
 ██▄ ▀  ▀ ▄██
█  ████████  █
 ▀█▀▄▄▀▀▄▄▀█▀
  ▀▄▄▄▀▀▄▄▄▀
`
}

func main() {
	addr := flag.StringP("addr", "a", "0.0.0.0:5000", "SSH server port")
	flag.Parse()

	s, err := wish.NewServer(
		wish.WithAddress(*addr),
		wish.WithHostKeyPath("storage/.ssh/id_fade"),
		// Accept any public key.
		ssh.PublicKeyAuth(func(ssh.Context, ssh.PublicKey) bool { return true }),
		// Do not accept password auth.
		ssh.PasswordAuth(func(ssh.Context, string) bool { return false }),
		wish.WithMiddleware(
			bubbletea.Middleware(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				return model{}, []tea.ProgramOption{
					tea.WithContext(sess.Context()),
					tea.WithAltScreen(),
				}
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
