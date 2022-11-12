package fswatch

import (
	"context"
	"regexp"
	"strings"

	"github.com/hinha/watchgo/config"
	"github.com/hinha/watchgo/core"
	"github.com/hinha/watchgo/utils"
)

// ProcessEvent construct
type ProcessEvent struct {
	ctx context.Context

	image *core.Image
	file  *core.File
}

// NewEvent cmd wrapper
func NewEvent(ctx context.Context) *ProcessEvent {
	return &ProcessEvent{
		ctx: ctx,
	}
}

func (p *ProcessEvent) Run(c chan string) {
	builder := core.NewBuilder()
	p.image = core.NewImageReader(builder)
	p.file = core.NewFileReader(builder)
	for i := 0; i < config.General.Worker; i++ {
		go p.process(c)
	}
}

func (p *ProcessEvent) process(event chan string) {
	reImage, err := regexp.Compile(core.Regexp())
	if err != nil {
		return
	}
	for {
		select {
		case evt := <-event:
			if strings.HasSuffix(evt, "~") {
				evt = evt[:len(evt)-1]
			}

			if utils.IgnoreExtension(evt) {
				continue
			}

			fsp := strings.SplitAfterN(evt, "/", -1)
			fxt := strings.Join(fsp[len(fsp)-1:], "")
			fd := strings.Join(fsp[:len(fsp)-1], "")
			var subFolder string
			if len(fd) > 1 {
				subFolder = fd[:len(fd)-1]
			} else {
				subFolder = ""
			}

			subPath := []string{subFolder, fxt}
			if reImage.MatchString(evt) {
				_ = p.image.Open(evt, subPath)
			} else {
				_ = p.file.Open(evt, subPath)
			}
		case <-p.ctx.Done():
			return
		}
	}
}
