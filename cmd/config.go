package main

import (
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// Struct to match the new YAML file structure
type Config struct {
	RootPath        string `yaml:"root"`
	ConfigPath      string `yaml:"config_path"`
	Parallel        int    `yaml:"parallel"`
	DownloadsTmpDir string `yaml:"tmp_dir"`
	//
	Download      map[string][]string `yaml:"download"`
	Downloaded    []string            `yaml:"downloaded"`
	downloadedMap map[string]bool     `yaml:"downloaded_map"`
}

func (c *Config) DownloadFinished(url string) bool {
	return c.downloadedMap[url]
}

func (c *Config) DownloadAppend(url string) (err error) {
	c.downloadedMap[url] = true
	c.Downloaded = append(c.Downloaded, url)
	return c.updateConfig()
}

func (c *Config) updateConfig() (err error) {
	// Ensure the directory exists
	if _, err := os.Stat(c.ConfigPath); os.IsNotExist(err) {
		os.MkdirAll(c.ConfigPath, os.ModePerm)
	}

	// Convert the map to a slice
	c.Downloaded = make([]string, 0, len(c.downloadedMap))
	for url := range c.downloadedMap {
		c.Downloaded = append(c.Downloaded, url)
	}

	// Save both the map and the list
	viper.Set("downloaded_map", c.downloadedMap)
	viper.Set("downloaded_list", c.Downloaded)

	configData, err := yaml.Marshal(viper.AllSettings())
	if err != nil {
		return err
	}

	err = os.WriteFile(c.ConfigPath, configData, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
