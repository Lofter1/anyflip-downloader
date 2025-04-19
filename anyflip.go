package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/schollz/progressbar/v3"
)

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

func (fb *flipbook) downloadImages(downloadFolder string, options downloadOptions) error {
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

	// Generate pages to download
	go func() {
		for page := range fb.pageCount {
			downloadPages <- page
		}
		close(downloadPages)
	}()

	downloadWorker := func() {
		defer wg.Done()
		for page := range downloadPages {
			if err := fb.downloadPage(page, downloadFolder, options); err != nil {
				downloadErrors <- err
			} else {
				bar.Add(1)
			}
		}
	}

	wg.Add(options.threads)
	for range options.threads {
		go downloadWorker()
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

func (fb *flipbook) downloadPage(page int, folder string, options downloadOptions) error {
	downloadURL := fb.pageURLs[page]

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= options.retries; attempt++ {
		resp, err = http.Get(downloadURL)
		if err == nil {
			break
		}
		time.Sleep(options.retryDelay)
	}

	if err != nil {
		return fmt.Errorf("download failed for %s after %d attempts: %w", downloadURL, options.retries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 response from %s: %s", downloadURL, resp.Status)
	}

	filename := fmt.Sprintf("%04d%s", page, path.Ext(downloadURL))
	filepath := path.Join(folder, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
