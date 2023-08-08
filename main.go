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
	"regexp"
	"strconv"
	"strings"

	"github.com/br3w0r/goitopdf/itopdf"
	"github.com/schollz/progressbar/v3"
)

var title string
var pageCount int
var insecure bool

func init() {
	flag.StringVar(&title, "title", "", "Specifies the name of the generated PDF document (uses book title if not specified)")
	flag.BoolVar(&insecure, "insecure", false, "Skip certificate validation")
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
	err = prepareDownload(anyflipURL)
	if err != nil {
		log.Fatal(err)
	}

	downloadFolder := title
	outputFile := title + ".pdf"

	err = downloadImages(anyflipURL, pageCount, downloadFolder)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Converting to pdf")
	err = createPDF(outputFile, downloadFolder)
	if err != nil {
		log.Fatal(err)
	}

	os.RemoveAll(downloadFolder)
}

func prepareDownload(anyflipURL *url.URL) error {
	sanitizeURL(anyflipURL)
	configjs, err := downloadConfigJSFile(anyflipURL)
	if err != nil {
		return err
	}

	if title == "" {
		title, err = getBookTitle(configjs)
		if err != nil {
			title = path.Base(anyflipURL.String())
		}
	}

	pageCount, err = getPageCount(configjs)
	return err
}

func sanitizeURL(anyflipURL *url.URL) {
	bookURLPathElements := strings.Split(anyflipURL.Path, "/")
	anyflipURL.Path = path.Join("/", bookURLPathElements[1], bookURLPathElements[2])
}

func createPDF(outputFile string, imageDir string) error {
	pdf := itopdf.NewInstance()
	err := pdf.WalkDir(imageDir, nil)
	if err != nil {
		return err
	}
	err = pdf.Save(outputFile)
	if err != nil {
		return err
	}
	return nil
}

func downloadImages(url *url.URL, pageCount int, downloadFolder string) error {
	err := os.Mkdir(downloadFolder, os.ModePerm)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions(pageCount,
		progressbar.OptionFullWidth(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Downloading"),
	)
	downloadURL, err := url.Parse("https://online.anyflip.com")
	if err != nil {
		return err
	}

	for page := 1; page <= pageCount; page++ {
		downloadURL.Path = path.Join(url.Path, "files", "mobile", strconv.Itoa(page)+".jpg")
		response, err := http.Get(downloadURL.String())
		if err != nil {
			return err
		}

		if response.StatusCode != http.StatusOK {
			return errors.New("Received non-200 response: " + response.Status)
		}

		extension := path.Ext(downloadURL.String())
		filename := fmt.Sprintf("%04d%v", page, extension)
		file, err := os.Create(path.Join(downloadFolder, filename))
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, response.Body)
		if err != nil {
			return err
		}

		bar.Add(1)
	}
	fmt.Println()
	return nil
}

func getBookTitle(configjs string) (string, error) {
	r := regexp.MustCompile(`("?(bookConfig.)?bookTitle"?[=]"(.*?)")|"title":"(.*?)"`)
	match := r.FindString(configjs)

	if strings.Contains(match, "=") {
		match = strings.Split(match, "=")[1]
	} else if strings.Contains(match, ":") {
		match = strings.Split(match, ":")[1]
	} else {
		return "", errors.New("could not find book title")
	}
	match = strings.ReplaceAll(match, "\"", "")

	return match, nil
}

func getPageCount(configjs string) (int, error) {
	r := regexp.MustCompile("\"?(bookConfig\\.)?totalPageCount\"?[=:]\"?\\d+\"?")
	match := r.FindString(configjs)

	if strings.Contains(match, "=") {
		match = strings.Split(match, "=")[1]
	} else if strings.Contains(match, ":") {
		match = strings.Split(match, ":")[1]
	} else {
		return 0, errors.New("could not find page count")
	}

	match = strings.ReplaceAll(match, "\"", "")
	return strconv.Atoi(match)
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
