package main

import (
	"fmt"
	"io"
)

type ProgressReader struct {
	Index    int
	FileName string
	io.Reader
	Total   int64
	Current *int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	*pr.Current += int64(n)

	// Update progress
	percentage := float64(*pr.Current) / float64(pr.Total) * 100
	updateDownloadStatus(pr.Index, fmt.Sprintf("Downloading (%s)... %.2f%% complete", pr.FileName, percentage))

	return n, err
}
