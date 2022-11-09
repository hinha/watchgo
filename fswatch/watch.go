package fswatch

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
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
var intervalDuration = 1 * time.Minute

type FSWatcher struct {
	FChan chan notify.EventInfo
	Paths []string
	M     *sync.RWMutex

	log      *log.Logger
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
			for _, p := range w.Paths {
				w.syncFile(p)
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

	w.log = ctx.Value(Log).(*log.Logger)

	builder := core.NewBuilder(w.log)
	w.image = core.NewImageReader(builder)
	w.file = core.NewFileReader(builder)

	starTime := time.Now()
	for _, p := range w.Paths {
		w.syncFile(p)
		go watcherInit(w.FChan, p)
	}
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

func (w *FSWatcher) syncFile(path string) {
	drive := make(chan resultSync)
	driveErr := make(chan error, 1)
	w.hardDrive(drive, driveErr)

	mDrive := make(map[string]string)
	for r := range drive {
		if r.err != nil {
			w.log.Printf("Error hard drive %v\n", r.err)
			continue
		}
		mDrive[r.sum] = r.path
	}

	if err := <-driveErr; err != nil {
		w.log.Printf("Error fatal hard drive %v\n", err)
		return
	}

	local := make(chan resultSync)
	localErr := make(chan error, 1)
	w.localDrive(path, local, localErr)
	for r := range local {
		if r.err != nil {
			w.log.Printf("Error local drive %v\n", r.err)
			continue
		}

		if _, ok := mDrive[r.sum]; ok {
			continue
		}

		reImage, err := regexp.Compile(core.Regexp())
		if err != nil {
			continue
		}

		subPath := strings.SplitAfter(r.path, path)
		if reImage.MatchString(r.path) {
			if err := w.image.Open(r.path, subPath); err != nil {
				w.log.Printf("Error %v", err)
			}
		} else {
			if err := w.file.Open(r.path, subPath); err != nil {
				w.log.Printf("Error %v", err)
			}
		}
	}

	if err := <-localErr; err != nil {
		w.log.Printf("Error fatal local drive %v\n", err)
		return
	}

	return
}

func (w *FSWatcher) hardDrive(c chan resultSync, errc chan error) {
	dirPath := path.Join(config.FileSystemCfg.Backup.HardDrivePath, core.GetStaticBackupFolder())
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.Mkdir(dirPath, os.ModePerm); err != nil {
			w.log.Printf("Error %v", err)
			return
		}
	}
	go walkDir(w.syncDone, c, errc, dirPath)
}

func (w *FSWatcher) localDrive(path string, c chan resultSync, errc chan error) {
	go walkDir(w.syncDone, c, errc, path)
}

func walkDir(done <-chan struct{}, c chan resultSync, errc chan error, path string) {
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
