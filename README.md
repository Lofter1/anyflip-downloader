# anyflip-downloader

[![build](https://github.com/elchinbaba/anyflip-downloader/actions/workflows/build.yml/badge.svg)](https://github.com/elchinbaba/anyflip-downloader/actions/workflows/build.yml)
[![goreleaser](https://github.com/elchinbaba/anyflip-downloader/actions/workflows/release.yml/badge.svg)](https://github.com/elchinbaba/anyflip-downloader/actions/workflows/release.yml)

Download anyflip books as PDF

![Demo](/assets/demo.gif)

## Disclaimer

Only use this tool to download books that officially allow PDFs to be downloaded.

## Installation
You can install this tool in multiple ways. Using the installation script or the go install command.

The install scripts are the suggested installation method for most users. 

### Install scripts

#### Linux/MacOS
Open the terminal and execute
```sh
curl -L https://raw.githubusercontent.com/elchinbaba/anyflip-downloader/elchinbaba-patch-1/scripts/install.sh | /usr/bin/env bash
```

#### Windows
Open PowerShell and execute
```PowerShell
. { iwr -useb https://raw.githubusercontent.com/elchinbaba/anyflip-downloader/elchinbaba-patch-1/scripts/install.ps1 } | iex;
```

### Go install
For `go install`, the [go tools](https://go.dev/doc/install) are required.

```sh
go install github.com/elchinbaba/anyflip-downloader@latest
```

## Usage

```sh
anyflip-downloader <url to book>
```

### Set title manually

If you do not want to use the book title from anyflip, you can change it using the `-title` flag.

```sh
anyflip-downloader <url to book> -title <your book title>
```

### Specify temporary download folder path

The default temporary download folder path will be the title of the book. However, in certain situations, you might want to change the temporary download folder. For this, the `-temp-download-folder` flag exists. This folder will be deleted after a successful download.

```sh
anyflip-downloader <url to book> -temp-download-folder <temp folder name>
```

