package config

import (
	"context"
	"log"
	"path/filepath"
	"strings"

	"github.com/rjeczalik/notify"
)

// Watch starts watching the given file for changes, and returns a channel to get notified on.
// Errors are also passed through this channel: Receiving a nil from the channel indicates the file is updated.
func Watch(ctx context.Context, pathFile string) (<-chan string, error) {
	notifyChan := make(chan notify.EventInfo)
	defer notify.Stop(notifyChan)

	absfile, err := filepath.Abs(pathFile)
	if err != nil {
		return nil, err
	}

	go func() {
		basedir := filepath.Dir(absfile)
		basedir = filepath.Join(basedir, "/...")
		if err := notify.Watch(basedir, notifyChan, notify.Create|notify.Write); err != nil {
			log.Fatalf("watch path %s error: %s\n", basedir, err)
		}
	}()

	writech := make(chan string, 100)
	go func() {
		for {
			select {
			case e := <-notifyChan:
				if strings.ReplaceAll(e.Path(), "~", "") == absfile {
					handleNotify(ctx, writech, e.Path())
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return writech, nil
}

func handleNotify(ctx context.Context, ch chan<- string, val string) {
	// Something happened...
	select {
	case ch <- val:
	case <-ctx.Done():
		return
	}
}
