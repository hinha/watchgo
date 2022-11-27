package config

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"log"
	"path/filepath"
	"strings"
)

// Watch starts watching the given file for changes, and returns a channel to get notified on.
// Errors are also passed through this channel: Receiving a nil from the channel indicates the file is updated.
func Watch(ctx context.Context, pathFile string) (<-chan string, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	absfile, err := filepath.Abs(pathFile)
	if err != nil {
		return nil, err
	}

	basedir := filepath.Dir(absfile)
	if err = watcher.Add(basedir); err != nil {
		return nil, err
	}

	writech := make(chan string, 100)
	go func() {
		for {
			select {
			case <-ctx.Done():
				watcher.Close()
				return
			case err := <-watcher.Errors:
				log.Printf("read config error %v", err)
			case e := <-watcher.Events:
				if e.Op&(fsnotify.Create|fsnotify.Write) > 0 {
					if strings.ReplaceAll(e.Name, "~", "") == absfile {
						handleNotify(ctx, writech, e.Name)
					}
				}
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
