# Downloader Tool

## Introduction

The Downloader tool is a command-line application written in Go, designed to simplify the process of downloading files based on a YAML configuration. It supports downloading in parallel, making it efficient for batch downloading tasks.

## Installation

To install the Downloader tool, download the latest version from the [Releases](https://github.com/yurifrl/downloader/releases/latest) page. Choose the appropriate binary for your operating system and architecture.

After downloading, you may need to make the binary executable (for Linux and macOS):

```bash
chmod +x downloader
```

## Usage

To use the Downloader tool, you need a YAML configuration file specifying the files to download. Here's an example command:

```bash
./downloader .
```

For more detailed usage instructions, refer to the tool's help:

```bash
./downloader --help
```

### Help

```
Downloads files from a YAML configuration

Usage:
  downloader [directory] [flags]

Flags:
  -c, --config string         config file (default is $HOME/.config.yaml) (default "config.yaml")
  -d, --download-dir string   Download directory (default "downloads")
  -h, --help                  help for downloader
  -p, --parallel int          parallel parameter (default 3)
```

## Building from Source

If you prefer to build the Downloader tool from source, ensure you have Go installed and run:

```bash
git clone https://github.com/yurifrl/downloader.git
cd downloader
go build -o downloader ./cmd
```