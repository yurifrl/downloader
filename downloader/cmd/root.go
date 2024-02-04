/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var rootCmd = &cobra.Command{
	Use:   "downloader [directory]",
	Short: "Downloads files from a YAML configuration",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		downloadDir := args[0]
		configFile := filepath.Join(downloadDir, "config.yaml")

		var config Config
		readConfig(configFile, &config)
		config.DownloadedURLsFile = "download_list.txt"

		semaphore := make(chan struct{}, config.Parallel)
		var wg sync.WaitGroup

		// Initialize downloadStatuses with expected size for better concurrency management
		downloadStatuses = make([]string, len(config.Download)*config.Parallel) // Adjust size estimation as needed
		downloadIndex := 0                                                      // Keep track of the index for assigning to downloads

		for folder, urls := range config.Download {
			fullPath := filepath.Join(downloadDir, folder)
			if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
				fmt.Printf("Failed to create directory %s: %v\n", fullPath, err)
				continue
			}

			for _, url := range urls {
				if config.DownloadFinished(url) {
					fmt.Printf("URL already downloaded: %s\n", url)
					continue
				}

				wg.Add(1)
				semaphore <- struct{}{} // Acquire a slot

				go func(url, fullPath string, index int) {
					defer wg.Done()                // Signal this download is complete
					defer func() { <-semaphore }() // Release the slot
					if err := downloadFile(url, config.DownloadsTmpDir, index); err != nil {
						fmt.Printf("Failed to download %s: %v\n", url, err)
					}
					// TODO unpack
				}(url, fullPath, downloadIndex)

				downloadIndex++ // Increment the index for the next download

				// TODO ENABLE THIS
				// config.DownloadAppend(url)
			}
		}

		wg.Wait()
		fmt.Println("All downloads and unpacking completed.")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.downloader.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Global variable to keep track of download statuses
var downloadStatuses []string
var statusLock sync.Mutex

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

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	*pr.Current += int64(n)

	// Update progress
	percentage := float64(*pr.Current) / float64(pr.Total) * 100
	updateDownloadStatus(pr.Index, fmt.Sprintf("Downloading (%s)... %.2f%% complete", pr.FileName, percentage))

	return n, err
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
	fmt.Print("\033[H\033[2J") // Clear the screen
	for _, s := range downloadStatuses {
		fmt.Println(s)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type ProgressReader struct {
	Index    int
	FileName string
	io.Reader
	Total   int64
	Current *int64
}

func readConfig(filePath string, config *Config) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		os.Exit(1)
	}
	if err := yaml.Unmarshal(file, config); err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		os.Exit(1)
	}
}

// / Struct to match the new YAML file structure
type Config struct {
	Download           map[string][]string `yaml:"download"`
	Parallel           int                 `yaml:"parallel"`
	DownloadedURLsFile string              `yaml:"downloaded_file"`
	DownloadsTmpDir    string              `yaml:"downloades_tmp_dir"`
}

func (c *Config) DownloadFinished(url string) bool {
	file, err := os.Open(c.DownloadedURLsFile)
	if err != nil {
		// Handle error (file might not exist which is okay on first run)
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == url {
			return true
		}
	}
	return false
}

func (c *Config) AppendURLToFile(url string) error {
	file, err := os.OpenFile(c.DownloadedURLsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(url + "\n")
	return err
}
