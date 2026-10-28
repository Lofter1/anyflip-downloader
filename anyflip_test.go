package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDownloadImagesResumesAtStartPage(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		_, _ = fmt.Fprint(w, "image"+r.URL.Path)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	downloadFolder := t.TempDir()
	for page, content := range []string{"page 1", "page 2"} {
		filename := filepath.Join(downloadFolder, fmt.Sprintf("%04d.jpg", page))
		if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	flipbook := &flipbook{
		URL:       serverURL,
		pageCount: 4,
		pageURLs: []string{
			server.URL + "/page0.jpg",
			server.URL + "/page1.jpg",
			server.URL + "/page2.jpg",
			server.URL + "/page3.jpg",
		},
	}
	err = flipbook.downloadImages(downloadFolder, downloadOptions{
		threads:   1,
		retries:   0,
		startPage: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"/page2.jpg", "/page3.jpg"}; !reflect.DeepEqual(requests, want) {
		t.Fatalf("requested pages = %v, want %v", requests, want)
	}

	for page, want := range map[int]string{
		0: "page 1",
		1: "page 2",
		2: "image/page2.jpg",
		3: "image/page3.jpg",
	} {
		filename := filepath.Join(downloadFolder, fmt.Sprintf("%04d.jpg", page))
		got, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("page %d contents = %q, want %q", page+1, got, want)
		}
	}
}

func TestDownloadImagesRejectsStartPageOutsideBook(t *testing.T) {
	flipbook := &flipbook{pageCount: 2}
	err := flipbook.downloadImages(t.TempDir(), downloadOptions{startPage: 2})
	if err == nil {
		t.Fatal("expected an out-of-range start page error")
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "book URL", input: "/book/flipbook/", want: "/book/flipbook"},
		{name: "URL with extra path", input: "/book/flipbook/files/", want: "/book/flipbook"},
		{name: "missing flipbook ID", input: "/book/", wantErr: true},
		{name: "empty path", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedURL := &url.URL{Path: tt.input}
			err := sanitizeURL(parsedURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("sanitizeURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && parsedURL.Path != tt.want {
				t.Errorf("sanitized path = %q, want %q", parsedURL.Path, tt.want)
			}
		})
	}

	if err := sanitizeURL(nil); err == nil {
		t.Error("sanitizeURL(nil) returned no error")
	}
}

func TestRunWithoutURLPrintsUsageAndReturns(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldFlagSet, oldUsage, oldStderr := flag.CommandLine, flag.Usage, os.Stderr
	defer func() {
		flag.CommandLine = oldFlagSet
		flag.Usage = oldUsage
		os.Stderr = oldStderr
	}()

	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	testFlagSet.SetOutput(write)
	usageCalled := false
	flag.Usage = func() {
		usageCalled = true
		_, _ = fmt.Fprintln(write, "usage")
	}
	flag.CommandLine = testFlagSet
	os.Stderr = write

	run()
	if err := write.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if !usageCalled {
		t.Fatal("run() did not print usage")
	}
	if !strings.Contains(string(output), "error: no URL provided") {
		t.Errorf("run() output = %q, want missing-URL message", output)
	}
	if !strings.Contains(string(output), "usage") {
		t.Errorf("run() output = %q, want usage text", output)
	}
}
