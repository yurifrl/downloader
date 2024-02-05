/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/k0kubun/pp/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Global variable to keep track of download statuses
var downloadStatuses []string
var statusLock sync.Mutex
var config Config

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	pp.Println("Begining...")

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("download-dir", "d", "downloads", "Download directory")
	rootCmd.PersistentFlags().StringP("config", "c", "config.yaml", "config file (default is $HOME/.config.yaml)")
	rootCmd.PersistentFlags().IntP("parallel", "p", 3, "parallel parameter")

	viper.BindPFlag("parallel", rootCmd.PersistentFlags().Lookup("parallel"))
	viper.BindPFlag("download_dir", rootCmd.PersistentFlags().Lookup("download-dir"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "downloader [directory]",
	Short: "Downloads files from a YAML configuration",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// pp.Println("Config ===>", config)

		if err := config.DownloadAppend("foo"); err != nil {
			log.Fatal(err)
		}

		semaphore := make(chan struct{}, config.Parallel)
		var wg sync.WaitGroup

		// Initialize downloadStatuses with expected size for better concurrency management
		downloadStatuses = make([]string, len(config.Download)*config.Parallel) // Adjust size estimation as needed
		downloadIndex := 0                                                      // Keep track of the index for assigning to downloads
		// Map to track directories that need to be created
		missingDirs := make(map[string]struct{})

		for folder, urls := range config.Download {
			fullPath := filepath.Join(config.DownloadsDir, folder)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				// Instead of logging, add to map
				missingDirs[fullPath] = struct{}{}
				continue
			}

			for _, url := range urls {
				if config.DownloadFinished(url) {
					log.Printf("URL already downloaded: %s\n", url)
					continue
				}

				wg.Add(1)
				semaphore <- struct{}{} // Acquire a slot

				go func(url, fullPath string, index int) {
					defer wg.Done()                // Signal this download is complete
					defer func() { <-semaphore }() // Release the slot
					if err := downloadFile(url, fullPath, index); err != nil {
						log.Printf("Failed to download %s: %v\n", url, err)
					}
					config.DownloadAppend(url)
					// TODO unpack
				}(url, fullPath, downloadIndex)

				downloadIndex++ // Increment the index for the next download
			}
		}

		// If there are missing directories, print the mkdir command
		if len(missingDirs) > 0 {
			var dirs []string
			for dir := range missingDirs {
				dirs = append(dirs, strconv.Quote(dir))
			}
			log.Printf("The following folders do not exist and will be created: \n%s\n", strings.Join(dirs, " "))
			log.Printf("mkdir -p %s\n", strings.Join(dirs, " "))
		}

		wg.Wait()
		log.Println("End.")
	},
}

// ExtractFileName extracts and decodes the file name from the URL.
func extractFileName(urlStr string) string {
	// Split the URL on "/" and get the last part
	parts := strings.Split(urlStr, "/")
	encodedFileName := parts[len(parts)-1]
	// Decode the percent-encoded filename
	decodedFileName, err := url.QueryUnescape(encodedFileName)
	if err != nil {
		// Handle the error or return the encodedFileName if decoding fails
		return encodedFileName
	}
	return decodedFileName
}

func downloadFile(urlStr, folderPath string, index int) (err error) {

	fileName := extractFileName(urlStr)
	filePath := filepath.Join(folderPath, fileName)

	log.Println("Starting download:", fileName)

	// Check if the file exists and how much has been downloaded
	fileInfo, err := os.Stat(filePath)
	var currentSize int64 = 0
	if err == nil {
		currentSize = fileInfo.Size()
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to create file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return
	}

	if currentSize > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(currentSize, 10)+"-")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	totalSize := currentSize
	if resp.Header.Get("Content-Range") != "" {
		parts := strings.Split(resp.Header.Get("Content-Range"), "/")
		totalSize, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			log.Println("Failed to parse total size from Content-Range")
			totalSize = currentSize // fallback to currentSize if cannot parse
		}
	} else if resp.Header.Get("Content-Length") != "" && currentSize == 0 {
		totalSize, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			log.Println("Failed to parse total size from Content-Length")
			totalSize = 0 // fallback to unknown size
		}
	}

	progressReader := &ProgressReader{
		Index:    index,
		FileName: fileName,
		Reader:   resp.Body,
		Total:    totalSize,
		Current:  &currentSize,
	}

	// Append the new data to the partial file
	_, err = io.Copy(file, progressReader)
	if err != nil {
		return err
	}

	log.Println("Completed download:", urlStr)
	return nil
}

func updateDownloadStatus(index int, status string) {
	statusLock.Lock()
	defer statusLock.Unlock()

	// Ensure the slice is large enough
	for len(downloadStatuses) <= index {
		downloadStatuses = append(downloadStatuses, "")
	}

	// Update the specific download status
	downloadStatuses[index] = status

	// Clear the screen and redraw statuses
	log.Print("\033[H\033[2J") // Clear the screen
	for _, s := range downloadStatuses {
		log.Println(s)
	}
}
