package fswatch

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/hinha/watchgo/config"
	"github.com/hinha/watchgo/core"
	"github.com/hinha/watchgo/logger"
	"github.com/hinha/watchgo/utils"
)

// intervalDuration sync every 30 minutes.
var intervalDuration = 30 * time.Minute

type FSWatcher struct {
	w      *fsnotify.Watcher
	Events chan fsnotify.Event

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

func (w *FSWatcher) FSWatcherStart(ctx context.Context, watch *fsnotify.Watcher) {
	w.w = watch

	w.syncDone = make(chan struct{})
	defer close(w.syncDone)

	builder := core.NewBuilder()
	w.image = core.NewImageReader(builder)
	w.file = core.NewFileReader(builder)

	starTime := time.Now()
	for i, p := range config.FileSystemCfg.Paths {
		w.syncFile(p, i)
		//go watcherInit(w.FChan, p)
		go watcherInit(ctx, w.w, p)
	}
	logger.Debug().Dur("duration", time.Since(starTime)).Msg("scanning complete")
	go janitor(ctx, w, time.Since(starTime))
}

func (w *FSWatcher) FSWatcherStop() {
	if err := w.w.Close(); err != nil {
		log.Fatal(err)
	}
}

// watcherInit.
func watcherInit(ctx context.Context, w *fsnotify.Watcher, path string) {
	dirs := func(root string) ([]string, error) {
		var folders []string
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				folders = append(folders, path)
			}
			return nil
		})
		return folders, err
	}

	list, err := dirs(path)
	if err != nil {
		log.Fatalf("walk dir %s", err)
	}

	// sync dir never stop initial watcher
	go func() {
		interval := 3 * time.Second
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				ticker.Stop()

				list, err := dirs(path)
				if err != nil {
					ticker.Stop()
					return
				}

				for _, f := range list {
					if err := w.Add(f); err != nil {
						log.Fatalf("watch path %s error: %s\n", path, err)
					}
				}

				// reset interval
				ticker = time.NewTicker(interval)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	for _, f := range list {
		if err := w.Add(f); err != nil {
			log.Fatalf("watch path %s error: %s\n", path, err)
		}
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
	defer close(driveErr)
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

	local := make(chan resultSync, config.General.WorkerBuffer)
	localErr := make(chan error, 1)
	defer close(localErr)

	w.localDrive(path, index, local, localErr)
	for work := 0; work < config.General.Worker; work++ {
		go func(id int, jobs <-chan resultSync) {
			for r := range jobs {
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
							countDuplicate++
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
						continue
					}
				} else {
					if err := w.file.Open(r.path, subPath); err != nil {
						continue
					}
				}
			}
		}(work, local)
	}

	if err := <-localErr; err != nil {
		logger.Error().Err(err).Msg("fatal local drive")
		return
	}
}

func (w *FSWatcher) hardDrive(c chan resultSync, errc chan error) {
	dirPath := path.Join(config.FileSystemCfg.Backup.HardDrivePath, config.GetStaticBackupFolder())
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		_ = os.Mkdir(dirPath, 0700)
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

				size := utils.ByteSize(info.Size())
				maxSize := utils.ByteSize(config.FileSystemCfg.MaxFileSize) * utils.MB
				if size >= maxSize {
					logger.Error().Str("path", path).Err(fmt.Errorf("size limit %s, of maximum %s", size.String(), maxSize.String())).Msg("local drive")
					return nil
				}
			}

			wg.Add(1)
			go func() {
				fos, err := os.Open(path)
				defer fos.Close()
				if err != nil {
					c <- resultSync{"", "", err}
					return
				}
				reader := bufio.NewReader(fos)

				hash := sha1.New()
				io.Copy(hash, reader)

				sum := hash.Sum(nil)
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
