package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type forwarder struct {
	url           string
	userAgent     string
	contentType   string
	authorization string
}

var httpClient = &http.Client{
	//Transport: &http.Transport{
	//    DisableCompression:    true, // accept/ignore Accept-Encoding: gzip
	//}
	Timeout: time.Second * 20,
}

func (f *forwarder) post(body io.Reader) (err error) {

	req, err := http.NewRequest("POST", f.url, body)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Content-Type", f.contentType)
	if f.authorization != "" {
		req.Header.Set("Authorization", f.authorization)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
	if (resp.StatusCode / 100) != 2 { // 2XX
		err = fmt.Errorf("post %s: status %d (%s)", f.url, resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return
}
