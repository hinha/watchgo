package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const AppName = "watch-go"

var (
	cfg  config
	File string

	Debug         bool
	General       = &cfg.General
	FileSystemCfg = &cfg.FileSystem
)

type config struct {
	General struct {
		Worker       int    `yaml:"worker"`
		WorkerBuffer int    `yaml:"worker_buffer"`
		EventBuffer  int    `yaml:"event_buffer"`
		Verbose      bool   `yaml:"verbose"`
		ErrorLog     string `yaml:"error_log"`
		InfoLog      string `yaml:"info_log"`
		PidFile      string `yaml:"pid_file"`
	} `yaml:"general"`
	FileSystem FileSystemConfig `yaml:"file_system"`
}

type FileSystemConfig struct {
	Paths       []string       `yaml:"paths"`
	Compress    CompressConfig `yaml:"compress"`
	MaxFileSize int64          `yaml:"max_file_size"`
	Backup      struct {
		HardDrivePath string   `yaml:"hard_drive_path"`
		Prefix        []string `yaml:"prefix"`
	} `yaml:"backup"`
}

type CompressConfig struct {
	Enabled bool `yaml:"enabled"`
	Quality int  `yaml:"quality"`
}

// LoadConfig Read and parse config file.
func LoadConfig(configFile string) (error error) {
	error = nil
	filename, err := filepath.Abs(configFile)
	if err != nil {
		log.Printf("[%s] error: %s fail find config\n", AppName, err)
		return err
	}

	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		log.Printf("[%s] error: %s parse from file %s\n", AppName, err, filename)
		return err
	}
	log.Printf("load settings √\n")
	return error
}

// ReloadConfig parse yml config
func ReloadConfig() error {
	filename, err := filepath.Abs(File)
	if err != nil {
		return fmt.Errorf("can not be reloaded, filepath Abs error: %s", err.Error())
	}
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("can not be reloaded, can not read yaml-File: %s", err.Error())
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return fmt.Errorf("[%s] error: %s parse from file %s\n", AppName, err, filename)
	}
	log.Printf("Config file re-load: %s", filename)
	return nil
}
