package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	watch     bool
	reindex   bool
	httpProxy string
	highlight bool
)

func init() {
	flag.BoolVar(&highlight, "highlight", false, "-highlight highlight result.")
	flag.BoolVar(&watch, "watch", true, "-watch watch bookmark changes and sync.")
	flag.BoolVar(&reindex, "reindex", false, "-reindex clear old and init.")
	flag.StringVar(&httpProxy, "http_proxy", "", "-http_proxy=http://127.0.0.1:7890")
}

func main() {
	flag.Parse()
	initHttpClient()
	InitDict()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	ch := make(chan struct{})
	go func() {
		<-done
		close(ch)
	}()
	inited := false
	_, jsonFileErr := os.Stat(jsonFile)
	_, bleveErr := os.Stat(bleveDir)
	if (os.IsNotExist(jsonFileErr) || os.IsNotExist(bleveErr)) || reindex {
		Info("init bookmark index.")
		os.Remove(jsonFile)
		os.RemoveAll(bleveDir)
		initBookmark(ch)
		inited = true
	}
	if watch {
		go watchChange(ch, inited)
	}
	go search(ch)
	<-ch
	time.Sleep(300 * time.Millisecond)
	Infof("ByeBye!")
}
