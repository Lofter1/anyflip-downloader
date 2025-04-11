package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
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

type flipbook struct {
	URL       *url.URL
	title     string
	pageCount int
	pageURLs  []string
}

type donwloadOptions struct {
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
}

func main() {
	flag.Parse()
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

	err = flipbook.downloadImages(tempDownloadFolder, donwloadOptions{threads: donwloadThreads, retries: downloadRetries, retryDelay: downloadRetryDelay})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Converting to pdf")
	err = createPDF(outputFile, tempDownloadFolder)
	if err != nil {
		log.Fatal(err)
	}

	if !keepDownloadFolder {
		os.RemoveAll(tempDownloadFolder)
	}
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s [OPTIONS] <url>\n", os.Args[0])
	fmt.Fprintf(w, "Options:\n")
	flag.PrintDefaults()
}

func prepareDownload(anyflipURL *url.URL) (*flipbook, error) {
	var newFlipbook flipbook

	sanitizeURL(anyflipURL)
	newFlipbook.URL = anyflipURL

	configjs, err := downloadConfigJSFile(anyflipURL)
	if err != nil {
		return nil, err
	}

	if title == "" {
		title, err = getBookTitle(configjs)
		if err != nil {
			title = path.Base(anyflipURL.String())
		}
	}

	newFlipbook.title = govalidator.SafeFileName(title)
	newFlipbook.pageCount, err = getPageCount(configjs)
	pageFileNames := getPageFileNames(configjs)

	downloadURL, _ := url.Parse("https://online.anyflip.com/")
	println(newFlipbook.URL.String())
	if len(pageFileNames) == 0 {
		for i := 1; i <= newFlipbook.pageCount; i++ {
			downloadURL.Path = path.Join(newFlipbook.URL.Path, "files", "mobile", strconv.Itoa(i)+".jpg")
			newFlipbook.pageURLs = append(newFlipbook.pageURLs, downloadURL.String())
		}
	} else {
		for i := 0; i < newFlipbook.pageCount; i++ {
			downloadURL.Path = path.Join(newFlipbook.URL.Path, "files", "large", pageFileNames[i])
			newFlipbook.pageURLs = append(newFlipbook.pageURLs, downloadURL.String())
		}
	}

	return &newFlipbook, err
}

func sanitizeURL(anyflipURL *url.URL) {
	bookURLPathElements := strings.Split(anyflipURL.Path, "/")
	anyflipURL.Path = path.Join("/", bookURLPathElements[1], bookURLPathElements[2])
}

func createPDF(outputFile string, imageDir string) error {
	outputFile = strings.ReplaceAll(outputFile, "'", "")
	outputFile = strings.ReplaceAll(outputFile, "\\", "")
	outputFile = strings.ReplaceAll(outputFile, ":", "")

	if _, err := os.Stat(outputFile); err == nil {
		fmt.Printf("Output file %s already exists", outputFile)
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

	impConf := pdfcpu.DefaultImportConfig()
	err = api.ImportImagesFile(imagePaths, outputFile, impConf, nil)

	return err
}

func (fb *flipbook) downloadImages(downloadFolder string, options donwloadOptions) error {
	err := os.Mkdir(downloadFolder, os.ModePerm)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions(fb.pageCount,
		progressbar.OptionFullWidth(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Downloading"),
	)

	downloadPages := make(chan int)
	downloadErrors := make(chan error)

	var wg sync.WaitGroup
	wg.Add(options.threads)

	// Generate pages to download
	go func() {
		for page := 0; page < fb.pageCount; page++ {
			downloadPages <- page
		}
		close(downloadPages)
	}()

	for thread := 0; thread < options.threads; thread++ {
		go func() {
			defer wg.Done()

			for page := range downloadPages {
				func() {
					downloadURL := fb.pageURLs[page]

					var response *http.Response
					var err error

					for attempt := 1; attempt <= options.retries; attempt++ {
						response, err = http.Get(downloadURL)
						if err == nil {
							break
						}

						if attempt < options.retries {
							time.Sleep(options.retryDelay)
						} else {
							downloadErrors <- fmt.Errorf("download failed after %d attempts: %w", options.retries, err)
							return
						}
					}
					defer response.Body.Close()

					if response.StatusCode != http.StatusOK {
						downloadErrors <- fmt.Errorf("during download from %s received non-200 response: %s", downloadURL, response.Status)
						return
					}

					extension := path.Ext(downloadURL)
					filename := fmt.Sprintf("%04d%v", page, extension)
					file, err := os.Create(path.Join(downloadFolder, filename))
					if err != nil {
						downloadErrors <- err
						return
					}
					defer file.Close()

					_, err = io.Copy(file, response.Body)
					if err != nil {
						downloadErrors <- err
						return
					}

					bar.Add(1)
				}()
			}
		}()
	}

	// Wait for all downloads to finish
	go func() {
		wg.Wait()
		close(downloadErrors)
	}()

	var errors []error
	for err := range downloadErrors {
		fmt.Printf("Error occured: %e\n", err)
		errors = append(errors, err)
	}

	fmt.Println()
	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

func downloadConfigJSFile(bookURL *url.URL) (string, error) {
	configjsURL, err := url.Parse("https://online.anyflip.com")
	if err != nil {
		return "", err
	}
	configjsURL.Path = path.Join(bookURL.Path, "mobile", "javascript", "config.js")
	resp, err := http.Get(configjsURL.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("received non-200 response:" + resp.Status)
	}
	configjs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(configjs), nil
}
