package fswatch

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/hinha/watchgo/logger"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rjeczalik/notify"

	"github.com/hinha/watchgo/config"
	"github.com/hinha/watchgo/core"
	"github.com/hinha/watchgo/utils"
)

// intervalDuration sync every 30 minutes
var intervalDuration = 30 * time.Minute

type FSWatcher struct {
	FChan chan notify.EventInfo

	syncDone chan struct{}
	image    *core.Image
	file     *core.File
}

func janitor(ctx context.Context, w *FSWatcher, interval time.Duration) {
	w.syncDone = make(chan struct{})
	defer close(w.syncDone)

	startInterval := interval.Seconds() + intervalDuration.Seconds()
	done := make(chan bool)
	ticker := time.NewTicker(time.Duration(startInterval) * time.Second)

	for {
		select {
		case <-done:
			ticker.Stop()
			return
		case <-ticker.C:
			ticker.Stop()

			starTime := time.Now()
			for i, p := range config.FileSystemCfg.Paths {
				w.syncFile(p, i)
			}

			// reset interval
			ticker = time.NewTicker(time.Duration(time.Since(starTime).Seconds()+intervalDuration.Seconds()) * time.Second)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (w *FSWatcher) FSWatcherStart(ctx context.Context) {
	w.syncDone = make(chan struct{})
	defer close(w.syncDone)

	builder := core.NewBuilder()
	w.image = core.NewImageReader(builder)
	w.file = core.NewFileReader(builder)

	starTime := time.Now()
	for i, p := range config.FileSystemCfg.Paths {
		w.syncFile(p, i)
		go watcherInit(w.FChan, p)
	}
	logger.Debug().Dur("duration", time.Since(starTime)).Msg("scanning complete")
	go janitor(ctx, w, time.Since(starTime))
}

func (w *FSWatcher) FSWatcherStop() {
	notify.Stop(w.FChan)
}

func (w *FSWatcher) FSWatcherRestart(ctx context.Context) {
	w.FSWatcherStop()
	w.FSWatcherStart(ctx)
}

// watcherInit
func watcherInit(ec chan notify.EventInfo, path string) {
	path = filepath.Join(path, "/...")
	if err := notify.Watch(path, ec, notify.Create); err != nil {
		log.Fatalf("watch path %s error: %s\n", path, err)
	}
}

// A resultSync is the product of reading and summing a file using MD5.
type resultSync struct {
	path string
	sum  string
	err  error
}

func (w *FSWatcher) syncFile(path string, index int) {
	drive := make(chan resultSync)
	driveErr := make(chan error, 1)
	w.hardDrive(drive, driveErr)

	mDrive := make(map[string]string)
	for r := range drive {
		if r.err != nil {
			logger.Error().Err(r.err).Msg("hard drive")
			continue
		}
		mDrive[r.sum] = r.path
	}

	if err := <-driveErr; err != nil {
		logger.Error().Err(err).Msg("fatal hard drive")
		return
	}

	local := make(chan resultSync)
	localErr := make(chan error, 1)
	w.localDrive(path, index, local, localErr)
	for r := range local {
		if r.err != nil {
			logger.Error().Err(r.err).Msg("local drive")
			continue
		}

		if _, ok := mDrive[r.sum]; ok {
			continue
		} else {
			var countDuplicate int
			for _, v := range mDrive {
				if filepath.Base(v) == filepath.Base(r.path) {
					countDuplicate += 1
				}
			}

			if countDuplicate >= 1 {
				continue
			}
		}

		reImage, err := regexp.Compile(core.Regexp())
		if err != nil {
			continue
		}

		subPath := strings.SplitAfter(r.path, path)
		if reImage.MatchString(r.path) {
			if err := w.image.Open(r.path, subPath); err != nil {
				logger.Error().Err(err).Msg("image sync")
			}
		} else {
			if err := w.file.Open(r.path, subPath); err != nil {
				logger.Error().Err(err).Msg("file sync")
			}
		}
	}

	if err := <-localErr; err != nil {
		logger.Error().Err(err).Msg("fatal local drive")
		return
	}

	return
}

func (w *FSWatcher) hardDrive(c chan resultSync, errc chan error) {
	dirPath := path.Join(config.FileSystemCfg.Backup.HardDrivePath, core.GetStaticBackupFolder())
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.Mkdir(dirPath, os.ModePerm); err != nil {
			logger.Error().Err(err).Msg("hard drive create folder")
			return
		}
	}
	go walkDir(w.syncDone, c, errc, dirPath, 0, false)
}

func (w *FSWatcher) localDrive(path string, index int, c chan resultSync, errc chan error) {
	go walkDir(w.syncDone, c, errc, path, index, true)
}

func walkDir(done <-chan struct{}, c chan resultSync, errc chan error, path string, index int, runLocal bool) {
	var wg sync.WaitGroup
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if utils.IgnoreExtension(path) {
			return nil
		}

		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		if !info.IsDir() {
			if runLocal {
				_, after, _ := strings.Cut(path, config.FileSystemCfg.Paths[index])
				// start from .Folder/foo
				ok, _ := utils.IsHiddenFile(after[1:])
				if ok {
					return nil
				}
			}

			wg.Add(1)
			go func() {
				data, err := os.ReadFile(path)
				sum := md5.Sum(data)
				select {
				case c <- resultSync{path, hex.EncodeToString(sum[:]), err}:
				case <-done:
				}
				wg.Done()
			}()
		}

		// Abort the walk if done is closed.
		select {
		case <-done:
			return errors.New("walk canceled")
		default:
			return nil
		}
	})

	// Walk has returned, so all calls to wg.Add are done.  Start a
	// goroutine to close c once all the sends are done.
	go func() {
		wg.Wait()
		close(c)
	}()

	errc <- err
}
