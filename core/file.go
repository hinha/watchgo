package core

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hinha/watchgo/config"
	"github.com/hinha/watchgo/utils"
)

func NewFileReader(builder Builder) *File {
	return &File{builder: builder}
}

type File struct {
	builder Builder
}

func (i *File) Open(lPath string, subPath []string) error {
	folder := i.builder.createFolder(subPath)
	if folder == "" {
		return fmt.Errorf("error creating folder")
	}

	fi, err := os.Stat(lPath)
	if err != nil {
		return err
	}

	size := utils.ByteSize(fi.Size())
	maxSize := utils.ByteSize(config.FileSystemCfg.MaxFileSize) * utils.MB
	if size >= maxSize {
		return fmt.Errorf("size limits on the file %s of maximum, %s", fi.Name(), maxSize.String())
	}

	lPath = filepath.Clean(lPath)
	dstPath := filepath.Clean(path.Join(folder, fi.Name()))
	i.builder.copy(lPath, dstPath)

	return nil
}
