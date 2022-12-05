package core

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hinha/watchgo/config"
)

var (
	cmdJPG = "JPEG"
	cmdPNG = "PNG"
)

func NewImageReader(builder Builder) *Image {
	return &Image{builder: builder}
}

type Image struct {
	builder Builder
}

func (i *Image) Open(lPath string, subPath []string) error {
	folder := i.builder.createFolder(subPath)
	if folder == "" {
		return fmt.Errorf("error creating folder")
	}
	fi, _ := os.Stat(lPath)

	lPath = filepath.Clean(lPath)
	dstPath := filepath.Clean(path.Join(folder, fi.Name()))
	i.builder.copy(lPath, dstPath)

	interlace := cmdPNG
	if IsJpg.MatchString(lPath) {
		interlace = cmdJPG
	}

	if config.FileSystemCfg.Compress.Enabled {
		i.builder.compress(config.FileSystemCfg.Compress.Quality, dstPath, interlace)
	}

	return nil
}
