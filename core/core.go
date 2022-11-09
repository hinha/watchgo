package core

import "C"
import (
	"fmt"
	"github.com/hinha/watchgo/config"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const staticBackupFolder = "backup_test"

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

type builder struct {
	log *log.Logger
}

func (c *builder) createFolder(subPath []string) string {
	_, subFolder := subPath[0], subPath[1]
	if strings.HasPrefix(subFolder, "/") {
		// remove trailing slash
		subFolder = subFolder[1:]
	}

	// remove it file with extension abc.foo
	fsp := strings.SplitAfterN(subFolder, "/", -1)
	fd := strings.Join(fsp[:len(fsp)-1], "")
	if len(fd) > 1 {
		subFolder = fd[:len(fd)-1]
	} else {
		subFolder = ""
	}

	originPath := path.Join(config.FileSystemCfg.Backup.HardDrivePath, staticBackupFolder, subFolder)
	if err := os.MkdirAll(originPath, os.ModePerm); err != nil {
		c.log.Printf("creating folder err: %v\n", err)
		return ""
	}

	return originPath
}

func (c *builder) copy(srcPath, dstPath string) {
	sourceFileStat, _ := os.Stat(srcPath)
	if !sourceFileStat.Mode().IsRegular() {
		c.log.Printf("Error %s is not a regular file\n", srcPath)
		return
	}

	source, err := os.Open(srcPath)
	if err != nil {
		c.log.Printf("Copy file error: %s\n", err.Error())
		return
	}
	defer source.Close()

	destination, err := os.Create(dstPath)
	if err != nil {
		c.log.Printf("Copy file error: %s\n", err.Error())
		return
	}
	defer destination.Close()

	_, _ = io.Copy(destination, source)
	c.log.Printf("Copy file %s into %s was successfully\n", filepath.Base(srcPath), dstPath)
}

func (c *builder) compress(quality int, filePath, interlace string) {
	fi, err := os.Stat(filePath)
	if err != nil {
		c.log.Printf("Load file error: %s\n", err.Error())
		return
	}
	beforeSize := fi.Size()

	cmd := fmt.Sprintf("identify -format %s '%s'", "'%Q'", filePath)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		c.log.Printf("Error incorrect file name %s", filePath)
		return
	}

	qualityNum, _ := strconv.ParseInt(string(out), 10, 0)
	if int64(quality) >= qualityNum {
		c.log.Printf("File %s already compressed\n", filePath)
		return
	}

	cmd = fmt.Sprintf("convert '%s' -sampling-factor 4:2:0 -strip -quality %d -interlace %s -colorspace sRGB '%s'",
		filePath,
		quality,
		interlace,
		filePath)

	if _, err := exec.Command("bash", "-c", cmd).Output(); err != nil {
		c.log.Printf("Compress image error: %s\n", err.Error())
	}

	fl, _ := os.Stat(filePath)
	afterSize := fl.Size()
	c.log.Printf("Compress file is done, filesize before %d, after %d\n", beforeSize, afterSize)
	return
}

func NewBuilder(log *log.Logger) Builder {
	return &builder{log: log}
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
