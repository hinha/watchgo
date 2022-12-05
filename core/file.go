package core

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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

	lPath = filepath.Clean(lPath)
	dstPath := filepath.Clean(path.Join(folder, fi.Name()))
	i.builder.copy(lPath, dstPath)

	return nil
}
