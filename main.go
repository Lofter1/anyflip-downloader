package main

import (
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

func main() {

	extractTitle := flag.Bool("extrectTitle", true, "used to decide if legacy naming system is used or title is extracted from anyflip")
	customName := flag.String("customName", "", "used to set output file name, overides extractTitle")
	anyflipURL, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	bookURLPathElements := strings.Split(anyflipURL.Path, "/")
	// secect only 1st and 2nd element of url to avoid mobile on online.anyflip urls
	// as path starts with / offset index by 1
	anyflipURL.Path = path.Join("/", bookURLPathElements[1], bookURLPathElements[2])

	downloadFolder := path.Base(anyflipURL.String())
	outputFile := path.Base(anyflipURL.String()) + ".pdf"

	configjs, err := downloadConfigJSFile(anyflipURL)
	if err != nil {
		log.Fatal(err)
	}

	//use custom name for output
	if *customName != "" {
		outputFile = *customName
	}

	// use --extract_title to automatically rename pdf to it's title from anyflip, default true
	if *extractTitle && *customName == "" {
		of, err := getBookTitle(anyflipURL, configjs)
		if err != nil {
			log.Fatal(err)
		}
		// fallback to old naming
		if of != "" {
			outputFile = of + ".pdf"
		}
	}

	fmt.Println("Preparing to download")
	pageCount, err := getPageCount(anyflipURL, configjs)
	if err != nil {
		log.Fatal(err)
	}
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

func getBookTitle(url *url.URL, configjs string) (string, error) {
	r := regexp.MustCompile("\"?(bookConfig\\.)?bookTitle\"?=\"(.*?)\"")
	match := r.FindString(configjs)
	match = match[22 : len(match)-1]
	return match, nil
}

func getPageCount(url *url.URL, configjs string) (int, error) {

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
