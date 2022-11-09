package config

import (
	"context"
	"github.com/rjeczalik/notify"
	"log"
	"path/filepath"
	"strings"
)

// Watch starts watching the given file for changes, and returns a channel to get notified on.
// Errors are also passed through this channel: Receiving a nil from the channel indicates the file is updated.
func Watch(ctx context.Context, pathtofile string) (<-chan string, error) {
	notifyChan := make(chan notify.EventInfo)
	defer notify.Stop(notifyChan)

	absfile, err := filepath.Abs(pathtofile)
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
	//watcher, err := fsnotify.NewWatcher()
	//if err != nil {
	//	return nil, err
	//}
	//
	//absfile, err := filepath.Abs(pathtofile)
	//if err != nil {
	//	return nil, err
	//}
	//basedir := filepath.Dir(absfile)
	//
	//if err = watcher.Add(basedir); err != nil {
	//	return nil, err
	//}
	//
	//writech := make(chan error, 100)
	//
	//go func() {
	//	for {
	//		select {
	//		case <-ctx.Done():
	//			watcher.Close()
	//			return
	//
	//		case err := <-watcher.Errors:
	//			handleNotify(ctx, writech, err)
	//
	//		case e := <-watcher.Events:
	//			if e.Op&(fsnotify.Create|fsnotify.Write) > 0 {
	//				if e.Name == absfile {
	//					handleNotify(ctx, writech, nil)
	//				}
	//			}
	//		}
	//
	//	}
	//}()
	//
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
