package core

import (
	"github.com/hinha/watchgo/config"
	"os"
	"path"
	"path/filepath"
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
