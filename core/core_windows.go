//go:build windows

package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hinha/watchgo/config"
	"github.com/hinha/watchgo/logger"
)

func init() {
	logger.SetGlobalLogger(logger.New())
}

var IsJpg, _ = regexp.Compile(`^.*.(JPG|jpeg|JPEG|jpg)$`)

func Regexp() string {
	if len(config.FileSystemCfg.Backup.Prefix) > 0 && config.FileSystemCfg.Backup.Prefix[0] != "*" {
		prefix := fmt.Sprintf(`(%s).*.(JPG|jpeg|JPEG|jpg|png|PNG|pdf)$`, strings.Join(config.FileSystemCfg.Backup.Prefix, "|"))
		return prefix
	}
	return `^.*.(JPG|jpeg|JPEG|jpg|png|PNG|pdf)$`
}

type Builder interface {
	compress(quality int, imagePath, interlace string)
	createFolder(subPath []string) string
	copy(srcPath, dstPath string)
}
