package main

import (
	"compress/gzip"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"time"
)

var httpClient *client

type client struct {
	c *http.Client
}

func (c *client) GetResponseString(url string) string {
	if url != "" {
		get, err := httpClient.Get(url)
		if err != nil {
			Error(err)
			return ""
		}
		defer get.Body.Close()
		readAll, _ := io.ReadAll(get.Body)
		return string(readAll)
	}
	return ""
}

func (c *client) Get(url string) (resp *http.Response, err error) {
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Add("Accept", "*/*")
	r.Header.Add("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	r.Header.Add("Connection", "keep-alive")
	r.Header.Add("Connection", "keep-alive")
	r.Header.Add("Sec-Fetch-Dest", "empty")
	r.Header.Add("Sec-Fetch-Mode", "cors")
	r.Header.Add("Sec-Fetch-Site", "same-site")
	r.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36")
	r.Header.Add("Accept-Encoding", "gzip")

	do, err := c.c.Do(r)
	if err != nil {
		return nil, err
	}
	var reader io.ReadCloser
	switch do.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(do.Body)
		if err != nil {
			do.Body.Close()
			return nil, err
		}
		do.Body = reader
	default:
	}

	return do, err
}

func initHttpClient() {
	transport := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		TLSHandshakeTimeout: 0,
	}
	if httpProxy != "" {
		Infof("use http proxy: '%s'", httpProxy)
		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return url.Parse(httpProxy)
		}
	}
	httpClient = &client{c: &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}}
}
