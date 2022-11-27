package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hinha/watchgo/config"
	"github.com/hinha/watchgo/fswatch"
	"github.com/hinha/watchgo/logger"
	"log"
	"os"
)

func init() {
	// print help
	if len(os.Args) < 2 {
		log.Println(fmt.Sprintf("Usage: %s -options=param\n\n", config.AppName))
		flag.PrintDefaults()
		os.Exit(0)
	}

	flag.BoolVar(&config.Debug, "debug", false, "examples --debug=true")
	flag.StringVar(&config.File, "c", "/etc/watchgo/config.yml", "examples --c=config.yml")
	flag.Parse()

	if err := config.LoadConfig(config.File); err != nil {
		log.Fatalf("fatal open config file %s, error: %s\n", config.File, err)
	}

	logger.SetGlobalLogger(logger.New())
}

func main() {
	ctx := context.Background()
	ch, err := config.Watch(ctx, config.File)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case <-ch:
				if err := config.ReloadConfig(); err != nil {
					logger.Error().Err(err).Msg("Error reloading config")
				}
			}
		}
	}()

	c := make(chan fsnotify.Event, config.General.WorkerBuffer)
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatal().Err(err)
	}

	//fchan := make(chan notify.EventInfo, config.General.EventBuffer)
	done := make(chan struct{}, 1)
	defer close(done)

	fswatch.NewEvent(ctx).Run(c)

	watcher := &fswatch.FSWatcher{Events: watch.Events}

	watcher.FSWatcherStart(ctx, watch)
	defer watch.Close()

	// Process events
	go func() {
		for {
			select {
			case <-ctx.Done():
				done <- struct{}{}
				watch.Close()
				return
			case ev := <-watch.Events:
				c <- ev
			}
		}
	}()

	_, ok := <-done
	if ok {
		logger.Info(0).Msg("exit.")
	}
	os.Exit(0)
}
