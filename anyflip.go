package main

import (
	"fmt"
	"io"
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

	safeTitle := govalidator.SafeFileName(title)
	if safeTitle == "" {
		safeTitle = path.Base(anyflipURL.Path)
	}
	if safeTitle == "" || safeTitle == "." {
		safeTitle = "anyflip-download"
	}
	// Trim trailing dots/spaces - Windows strips these and causes "path not found"
	safeTitle = strings.Trim(strings.TrimSpace(safeTitle), ".")
	if safeTitle == "" {
		safeTitle = "anyflip-download"
	}
	newFlipbook.title = safeTitle
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
	// Use absolute path to avoid "path not found" on Windows when cwd has issues
	absFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return fmt.Errorf("invalid download path %q: %w", downloadFolder, err)
	}
	err = os.MkdirAll(absFolder, os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot create folder %q: %w", absFolder, err)
	}
	downloadFolder = absFolder

	bar := progressbar.NewOptions(fb.pageCount,
		progressbar.OptionFullWidth(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Downloading"),
	)
	defer bar.Close()

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
	downloadURL := cleanDownloadURL(fb.pageURLs[page])

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= options.retries; attempt++ {
		var req *http.Request
		req, err = http.NewRequest("GET", downloadURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Referer", fb.URL.String())
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err = http.DefaultClient.Do(req)
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
	filePath := filepath.Join(folder, filename)
	file, err := os.Create(filePath)
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

// cleanDownloadURL removes problematic sequences from the download URL
// and resolves any "../" path traversal by traversing up the directory structure.
func cleanDownloadURL(rawURL string) string {
	decoded, err := url.PathUnescape(rawURL)
	if err != nil {
		decoded = rawURL
	}
	// Normalize backslashes to forward slashes before parsing
	decoded = strings.ReplaceAll(decoded, "\\", "/")

	u, err := url.Parse(decoded)
	if err != nil {
		return decoded
	}

	// Clean the path: resolves ".." and "." segments and removes double slashes
	u.Path = path.Clean(u.Path)

	// Deduplicate consecutive identical path segments produced by sites that
	// generate URLs like "files/large/../files/mobile/" where resolving ".."
	// lands back in "files/" and then appends another "files/" segment.
	segments := strings.Split(u.Path, "/")
	deduped := segments[:0]
	for i, seg := range segments {
		if i > 0 && seg == segments[i-1] && seg != "" {
			continue
		}
		deduped = append(deduped, seg)
	}
	u.Path = strings.Join(deduped, "/")

	return u.String()
}
