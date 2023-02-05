package main

import (
	"errors"
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
	anyflipURL, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	downloadFolder := path.Base(anyflipURL.String())
	outputFile := path.Base(anyflipURL.String()) + ".pdf"

	fmt.Println("Preparing to download")
	pageCount, err := getPageCount(anyflipURL)
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

func getPageCount(url *url.URL) (int, error) {
	configjsURL, err := url.Parse("https://online.anyflip.com")
	if err != nil {
		return 0, err
	}
	configjsURL.Path = path.Join(url.Path, "mobile", "javascript", "config.js")
	resp, err := http.Get(configjsURL.String())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, errors.New("Received non-200 response: " + resp.Status)
	}

	configjs, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	r := regexp.MustCompile("bookConfig.totalPageCount=\"\\d+\"")
	match := r.FindString(string(configjs))
	match = strings.Split(match, "=")[1]
	match = strings.ReplaceAll(match, "\"", "")
	return strconv.Atoi(match)
}
