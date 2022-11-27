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

var (
	version string
	build   string
	commit  string
	author  string
	docs    string
)

func init() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v" || os.Args[1] == "ver") {
		printVersion()
		os.Exit(0)
	}

	flag.BoolVar(&config.Debug, "debug", false, "examples --debug=true")
	flag.StringVar(&config.File, "c", "/etc/watchgo/config.yml", "examples --c=config.yml")
	flag.Parse()

	// print help
	if len(os.Args) < 2 {
		log.Printf("Usage: %s -options=param\n\n", config.AppName)
		flag.PrintDefaults()
		os.Exit(0)
	}

	printVersion()

	if err := config.LoadConfig(config.File); err != nil {
		log.Fatalf("fatal open config file %s, error: %s\n", config.File, err)
	}

	logger.SetGlobalLogger(logger.New())
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := config.Watch(ctx, config.File)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
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

// printVersion program build data.
func printVersion() {
	fmt.Printf("Version: %s\nBuild Time: %s\nGit Commit Hash: %s\nAuthor: %s\nDocs: %s\n\n\n", version, build, commit, author, docs)
}
