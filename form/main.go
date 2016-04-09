package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
)

type theApp struct {
	debug bool
}

// App is the struct that holds all application related attributes
var App theApp

func errcheck(err error) {
	if err != nil {
		fmt.Printf("err = %v\n", err)
		os.Exit(1)
	}
}

func getFile(q string) {
	URL := "https://directory.faa.gov/appsPub/National/employeedirectory/faadir.nsf/SearchForm?OpenForm"
	hc := http.Client{}

	form := url.Values{}
	form.Add("__Click", "862570240055C5F3.c191ad9beca4086705256f6b00650208/$Body/0.1158")
	form.Add("FAP_LastName", fmt.Sprintf("%s*", q))
	form.Add("FAP_FirstName", "")
	// form.Add("WebQuery", ` ([LastName] CONTAINS "be*")  AND ([FORM] CONTAINS "Profile") AND NOT ([Deleted]   CONTAINS "1") AND NOT ([OptOut] CONTAINS "Y")" Type="Hidden"`)
	// req.PostForm = form
	req, err := http.NewRequest("POST", URL, bytes.NewBufferString(form.Encode()))
	errcheck(err)
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Add("Cookie", "BIGipServerpool_prd_directory.faa.gov_https=3200430747.47873.0000")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.80 Safari/537.36")
	req.Header.Add("Referer", "https://directory.faa.gov/appsPub/National/employeedirectory/faadir.nsf/SearchForm?OpenForm")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Accept-Language", "en-US,en;q=0.8")
	req.Header.Add("Origin", "https://directory.faa.gov")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Cache-Control", "max-age=0")
	req.Header.Add("Host", "directory.faa.gov")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(form.Encode())))

	if App.debug {
		dump, err := httputil.DumpRequest(req, false)
		errcheck(err)
		fmt.Printf("\n\ndumpRequest = %s\n", string(dump))
	}

	resp, err := hc.Do(req)
	errcheck(err)
	defer resp.Body.Close()

	if App.debug {
		dump, err := httputil.DumpResponse(resp, true)
		errcheck(err)
		fmt.Printf("\n\ndumpResponse = %s\n", string(dump))
	}

	// Check that the server actually sent compressed data
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		defer reader.Close()
	default:
		reader = resp.Body
	}

	fname := fmt.Sprintf("%s.html", q)
	f, err := os.Create(fname)
	errcheck(err)
	defer f.Close()
	io.Copy(f, reader)
	f.Sync()
}

func getAllFiles() {
	for i := 'a'; i <= 'z'; i++ {
		for j := 'a'; j < 'z'; j++ {
			q := fmt.Sprintf("%c%c", i, j)
			fmt.Printf("%s\n", q)
			getFile(q)
		}
	}
}

func main() {
	App.debug = false
	getAllFiles()
}
