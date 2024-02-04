/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
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
					if err := downloadFile(url, tempDownloadDir, index); err != nil {
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
