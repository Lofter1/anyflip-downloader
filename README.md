# anyflip-downloader

[![build](https://github.com/Lofter1/anyflip-downloader/actions/workflows/build.yml/badge.svg)](https://github.com/Lofter1/anyflip-downloader/actions/workflows/build.yml)
[![goreleaser](https://github.com/Lofter1/anyflip-downloader/actions/workflows/release.yml/badge.svg)](https://github.com/Lofter1/anyflip-downloader/actions/workflows/release.yml)

Download anyflip books as PDF

## Disclaimer

Only use this tool to download books that officially allow PDFs to be downloaded.

## Usage

```sh
$ anyflip-downloader <url to book>
```

### Set title manually

If you do not want to use the book title from anyflip, you can change it using the `-title` flag.

```sh
$ anyflip-downloader <url to book> -title <your book title>
```

### Specify temporary download folder path

The default temporary download folder path will be the title of the book. However, in certain situations, you might want to change the temporary download folder. For this, the `-temp-download-folder` flag exists. This folder will be deleted after a successful download.

```sh
$ anyflip-downloader <url to book> -temp-download-folder <temp folder name>
```

## Installation

You can either download the executable from the release page, install from source or install with the `go install` command, which is the recommended way, as it enables easy updating.

For `go install` and installation through source, the [go tools](https://go.dev/doc/install) are required.

To use the `go install` command, just run

```sh
$ go install github.com/Lofter1/anyflip-downloader@latest
```
