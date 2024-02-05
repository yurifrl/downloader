package main

import (
	"bufio"
	"log"
	"os"

	"github.com/spf13/viper"
)

// Struct to match the new YAML file structure
type Config struct {
	RootPath       string              `yaml:"root"`
	Parallel       int                 `yaml:"parallel"`
	ConfigPath     string              `yaml:"config"`
	DownloadsDir   string              `yaml:"downloads"`
	Download       map[string][]string `yaml:"download"`
	DownloadedFile string              `yaml:"downloaded_file"`
	downloadedMap  map[string]bool     `yaml:"downloaded_map"`
}

// cobra init
func initConfig() {
	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		log.Fatalf("Config not found: `%s`", cfgFile)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal(err)
	}

	log.Println("Using config file:", viper.ConfigFileUsed())

	// TODO: Find out why this is not comming from default
	config.DownloadsDir = "downloads"
	config.RootPath = "."
	config.ConfigPath = "config.yaml"
	config.DownloadedFile = "downloads.txt"

	if err := config.Init(); err != nil {
		log.Fatal(err)
	}
}

// Init loads the downloaded URLs from downloaded.txt into downloadedMap
func (c *Config) Init() (err error) {
	c.downloadedMap = make(map[string]bool)
	file, err := os.Open(c.DownloadedFile) // Attempt to open the file
	if err != nil {
		if os.IsNotExist(err) {
			return // File does not exist, so it's like starting fresh
		}
		return // Other errors
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()
		c.downloadedMap[url] = true
	}

	return scanner.Err()
}

func (c *Config) DownloadFinished(url string) bool {
	return c.downloadedMap[url]
}

// DownloadAppend appends a new URL to downloaded.txt if not already downloaded
func (c *Config) DownloadAppend(url string) error {
	if c.downloadedMap[url] {
		return nil // URL already downloaded, nothing to do
	}

	// Open file in append mode, create if not exists
	file, err := os.OpenFile(c.DownloadedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the new URL to the file
	if _, err := file.WriteString(url + "\n"); err != nil {
		return err
	}

	// Update the map
	c.downloadedMap[url] = true
	return nil
}
