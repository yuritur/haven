package tui

import (
	"fmt"
	"sync"
	"time"
)

type Spinner struct {
	msg   string
	stopC chan struct{}
	doneC chan struct{}
	once  sync.Once
}

func StartSpinner(msg string) *Spinner {
	s := &Spinner{
		msg:   msg,
		stopC: make(chan struct{}),
		doneC: make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *Spinner) run() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	i := 0
	for {
		select {
		case <-s.stopC:
			fmt.Print("\r\033[K")
			close(s.doneC)
			return
		case <-ticker.C:
			fmt.Printf("\r%s  %s", frames[i%len(frames)], s.msg)
			i++
		}
	}
}

func (s *Spinner) Stop() {
	s.once.Do(func() {
		close(s.stopC)
		<-s.doneC
	})
}
