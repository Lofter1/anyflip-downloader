package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/schollz/progressbar/v3"
)

var title string
var tempDownloadFolder string
var insecure bool
var keepDownloadFolder bool
var donwloadThreads int
var downloadRetries int
var downloadRetryDelay time.Duration
var chunkSize uint

type flipbook struct {
	URL       *url.URL
	title     string
	pageCount int
	pageURLs  []string
}

type downloadOptions struct {
	threads    int
	retries    int
	retryDelay time.Duration
}

func init() {
	flag.Usage = printUsage
	flag.StringVar(&tempDownloadFolder, "temp-download-folder", "", "Specifies the name of the temporary download folder")
	flag.StringVar(&title, "title", "", "Specifies the name of the generated PDF document (uses book title if not specified)")
	flag.BoolVar(&insecure, "insecure", false, "Skip certificate validation")
	flag.BoolVar(&keepDownloadFolder, "keep-download-folder", false, "Keep the temporary download folder instead of deleting it after completion")
	flag.IntVar(&donwloadThreads, "threads", 1, "Number of parallel download processes")
	flag.IntVar(&downloadRetries, "retries", 1, "Number of download retries")
	flag.DurationVar(&downloadRetryDelay, "waitretry", time.Second, "Wait time between download retries")
	flag.UintVar(&chunkSize, "chunksize", 10, "Amount of images converted at once. Higher amount amount will end in less write actions but more memory usage")
}

func main() {
	flag.Parse()
	run()
}

func run() {
	anyflipURL, err := url.Parse(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	if insecure {
		fmt.Println("You enabled insecure downloads. This disables security checks. Stay safe!")
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	fmt.Println("Preparing to download")
	flipbook, err := prepareDownload(anyflipURL)
	if err != nil {
		log.Fatal(err)
	}

	if tempDownloadFolder == "" {
		tempDownloadFolder = flipbook.title
	}
	outputFile := title + ".pdf"

	err = flipbook.downloadImages(
		tempDownloadFolder,
		downloadOptions{threads: donwloadThreads, retries: downloadRetries, retryDelay: downloadRetryDelay},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Converting to pdf")
	err = createPDF(outputFile, tempDownloadFolder, int(chunkSize))
	if err != nil {
		log.Fatal(err)
	}

	if !keepDownloadFolder {
		os.RemoveAll(tempDownloadFolder)
	}

	fmt.Println("Done")
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s [OPTIONS] <url>\n", os.Args[0])
	fmt.Fprintf(w, "Options:\n")
	flag.PrintDefaults()
}

func createPDF(outputFile string, imageDir string, chunkSize int) error {
	outputFile = strings.ReplaceAll(outputFile, "'", "")
	outputFile = strings.ReplaceAll(outputFile, "\\", "")
	outputFile = strings.ReplaceAll(outputFile, ":", "")

	if _, err := os.Stat(outputFile); err == nil {
		fmt.Printf("Output file %s already exists\n", outputFile)
		return nil
	}

	files, err := os.ReadDir(imageDir)
	if err != nil {
		return err
	}

	var imagePaths []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := filepath.Ext(file.Name())
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			imagePaths = append(imagePaths, filepath.Join(imageDir, file.Name()))
		}
	}

	if len(imagePaths) == 0 {
		return fmt.Errorf("no images found in path %s", imageDir)
	}

	bar := progressbar.NewOptions(len(imagePaths),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Converting"),
	)
	defer bar.Close()

	impConf := pdfcpu.DefaultImportConfig()

	for i := 0; i < len(imagePaths); i += chunkSize {
		end := min(i+chunkSize, len(imagePaths))

		imagePathChunk := imagePaths[i:end]
		err = api.ImportImagesFile(imagePathChunk, outputFile, impConf, nil)
		bar.Add(min(chunkSize, len(imagePaths)-i))
	}

	fmt.Println()
	return err
}
