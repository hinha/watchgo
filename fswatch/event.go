package fswatch

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"regexp"
	"strings"

	"github.com/hinha/watchgo/config"
	"github.com/hinha/watchgo/core"
	"github.com/hinha/watchgo/utils"
)

// ProcessEvent construct.
type ProcessEvent struct {
	ctx context.Context

	image *core.Image
	file  *core.File
}

// NewEvent cmd wrapper.
func NewEvent(ctx context.Context) *ProcessEvent {
	return &ProcessEvent{
		ctx: ctx,
	}
}

func (p *ProcessEvent) Run(event chan fsnotify.Event) {
	builder := core.NewBuilder()
	p.image = core.NewImageReader(builder)
	p.file = core.NewFileReader(builder)
	for i := 0; i < config.General.Worker; i++ {
		go p.process(event)
	}
}

func (p *ProcessEvent) process(event chan fsnotify.Event) {
	reImage, err := regexp.Compile(core.Regexp())
	if err != nil {
		return
	}
	for {
		select {
		case evt := <-event:
			if evt.Op&(fsnotify.Create) > 0 {
				if strings.HasSuffix(evt.Name, "~") {
					evt.Name = evt.Name[:len(evt.Name)-1]
				}

				if utils.IgnoreExtension(evt.Name) {
					continue
				}

				fsp := strings.SplitAfterN(evt.Name, "/", -1)
				fxt := strings.Join(fsp[len(fsp)-1:], "")
				fd := strings.Join(fsp[:len(fsp)-1], "")
				var subFolder string
				if len(fd) > 1 {
					subFolder = fd[:len(fd)-1]
				} else {
					subFolder = ""
				}

				subPath := []string{subFolder, fxt}
				if reImage.MatchString(evt.Name) {
					_ = p.image.Open(evt.Name, subPath)
				} else {
					_ = p.file.Open(evt.Name, subPath)
				}
			}
		case <-p.ctx.Done():
			return
		}
	}
}
