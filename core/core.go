package core

import "C"
import (
	"fmt"
	"github.com/hinha/watchgo/config"
	"github.com/hinha/watchgo/logger"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const staticBackupFolder = "Backup Files"

var (
	IsJpg, _ = regexp.Compile(`^.*.(JPG|jpeg|JPEG|jpg)$`)
)

type ByteSize float64

const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
)

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

type builder struct{}

func (c *builder) createFolder(subPath []string) string {
	dstFolder, subFolder := subPath[0], subPath[1]
	if strings.HasPrefix(subFolder, "/") {
		// remove trailing slash
		subFolder = subFolder[1:]
	}

	dfs := strings.Split(dstFolder, "/")
	dstFolder = dfs[len(dfs)-1:][0]

	// remove it file with extension abc.foo
	fsp := strings.SplitAfterN(subFolder, "/", -1)
	fd := strings.Join(fsp[:len(fsp)-1], "")
	if len(fd) > 1 {
		subFolder = fd[:len(fd)-1]
	} else {
		subFolder = ""
	}

	originPath := path.Join(config.FileSystemCfg.Backup.HardDrivePath, staticBackupFolder, dstFolder, subFolder)
	if err := os.MkdirAll(originPath, os.ModePerm); err != nil {
		logger.Error().Err(err).Msg("creating folder")
		return ""
	}

	return originPath
}

func (c *builder) copy(srcPath, dstPath string) {
	duration := time.Now()
	sourceFileStat, _ := os.Stat(srcPath)
	if !sourceFileStat.Mode().IsRegular() {
		logger.Error().Err(fmt.Errorf("error %s is not a regular file", srcPath)).Msg("")
		return
	}

	source, err := os.Open(srcPath)
	if err != nil {
		logger.Error().Err(err).Msg("source open file")
		return
	}
	defer source.Close()

	destination, err := os.Create(dstPath)
	if err != nil {
		logger.Error().Err(err).Msg("destination create file")
		return
	}
	defer destination.Close()

	_, _ = io.Copy(destination, source)
	logger.Info(time.Since(duration)).Msg(fmt.Sprintf("copy file %s into %s was successfully", filepath.Base(srcPath), dstPath))
}

func (c *builder) compress(quality int, filePath, interlace string) {
	duration := time.Now()
	fi, err := os.Stat(filePath)
	if err != nil {
		logger.Error().Err(err).Msg("load file")
		return
	}
	beforeSize := fi.Size()

	cmd := fmt.Sprintf("identify -format %s '%s'", "'%Q'", filePath)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		logger.Error().Err(err).Msg(fmt.Sprintf("incorrect file name %s", filePath))
		return
	}

	qualityNum, _ := strconv.ParseInt(string(out), 10, 0)
	if int64(quality) >= qualityNum {
		logger.Info(time.Since(duration)).Msg(fmt.Sprintf("file %s already compressed", filePath))
		return
	}

	cmd = fmt.Sprintf("convert '%s' -sampling-factor 4:2:0 -strip -quality %d -interlace %s -colorspace sRGB '%s'",
		filePath,
		quality,
		interlace,
		filePath)

	if _, err := exec.Command("bash", "-c", cmd).Output(); err != nil {
		logger.Error().Err(err).Msg("compress image")
	}

	fl, _ := os.Stat(filePath)
	afterSize := fl.Size()
	logger.Info(time.Since(duration)).Dur("duration", time.Since(duration)).Msg(fmt.Sprintf("compress file is done, filesize before %d, after %d", beforeSize, afterSize))
	return
}

func NewBuilder() Builder {
	return &builder{}
}

func GetStaticBackupFolder() string {
	return staticBackupFolder
}

func (b ByteSize) String() string {
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2fGB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.2fMB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.2fKB", b/KB)
	}
	return fmt.Sprintf("%.2fB", b)
}
