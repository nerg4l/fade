package main

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"time"
)

type soundServer struct {
	w  io.Writer
	c  chan soundMsg
	lc chan soundLoopMsg
}

func (as *soundServer) Start(ctx context.Context) {
	t := time.NewTicker(500 * time.Millisecond)

	tune := tunes["start"]
	defer t.Stop()
	for i := 0; true; i %= len(tune) {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			as.w.Write(tune[i])
			i++
		case name := <-as.c:
			tu, ok := tunes[string(name)]
			if !ok {
				break
			}
			as.w.Write(tu[0])
			for i := range tu[1:] {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					as.w.Write(tu[i])
				}
			}
			i = 0
		case name := <-as.lc:
			tu, ok := tunes[string(name)]
			if ok {
				i = 0
				tune = tu
			}
		}
	}
}

var tunes = map[string][][]byte{
	"start": {
		[]byte("C4 0.5 \n"),
		[]byte("C4 0.3 \n"),
		[]byte("D4 0.2 \n"),
		[]byte("E4 0.8 \n"),
	},
}

type soundMsg string
type soundLoopMsg string

func (as *soundServer) Update(msg tea.Msg) {
	switch msg := msg.(type) {
	case soundMsg:
		as.c <- msg
	case soundLoopMsg:
		as.lc <- msg
	}
}
