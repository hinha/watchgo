package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/hinha/watchgo/fswatch"
	"github.com/rjeczalik/notify"
	"log"
	"os"
	"sync"

	"github.com/hinha/watchgo/config"
)

func init() {
	// print help
	if len(os.Args) < 2 {
		fmt.Println(fmt.Sprintf("Usage: %s -options=param\n\n", config.AppName))
		flag.PrintDefaults()
		os.Exit(0)
	}

	flag.BoolVar(&config.Debug, "debug", false, "examples --debug=true")
	flag.StringVar(&config.ConfigFile, "c", "/etc/watchgo/config.yml", "examples --c=config.yml")
	flag.Parse()

	if err := config.LoadConfig(config.ConfigFile); err != nil {
		log.Fatalf("fatal open config file %s, error: %s\n", config.ConfigFile, err)
	}

	config.Logger = NewLogger(&LogConfig{
		AppName: config.AppName,
		Debug:   config.Debug,
		LogFile: config.General.InfoLog,
	})
}

func main() {
	ctx := context.Background()
	ctx = fswatch.Register(ctx, fswatch.Log, config.Logger)
	ch, err := config.Watch(ctx, config.ConfigFile)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case <-ch:
				if err := config.ReloadConfig(); err != nil {
					config.Logger.Printf("Error reloading config %v", err)
				}
			}
		}
	}()

	c := make(chan string, config.General.WorkerBuffer)
	fchan := make(chan notify.EventInfo, config.General.EventBuffer)
	done := make(chan struct{}, 1)

	fswatch.NewEvent(ctx).Run(c)

	watcher := &fswatch.FSWatcher{
		FChan: fchan,
		Paths: config.FileSystemCfg.Paths,
		M:     &sync.RWMutex{},
	}

	watcher.FSWatcherStart(ctx)
	defer notify.Stop(fchan)

	// Process events
	go func() {
		for {
			select {
			case ev := <-fchan:
				c <- ev.Path()
			case <-ctx.Done():
				return
			}
		}
	}()

	<-done
	log.Println("exit.")
}
