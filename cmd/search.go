package cmd

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	bleveDir = "/tmp/cache.bleve"
	jsonFile = "/tmp/bookmark.json"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "全文搜索书签内容",
	Run: func(cmd *cobra.Command, args []string) {
		_, jsonFileErr := os.Stat(jsonFile)
		_, bleveErr := os.Stat(bleveDir)
		if os.IsNotExist(jsonFileErr) || os.IsNotExist(bleveErr) {
			log.Println("目录不存在，现在进行初始化")
			os.Remove(jsonFile)
			os.Remove(bleveDir)
			initBookmark()
		}

		ch := make(chan struct{})
		done := make(chan os.Signal, 1)
		if watch {
			go watchChange(ch)
		}
		signal.Notify(done, os.Interrupt, syscall.SIGTERM)

		go func() {
			for {
				func() {
					fmt.Printf("请输入你要搜的内容：")
					q, _ := bufio.NewReader(os.Stdin).ReadString('\n')
					index, _ := bleve.Open(bleveDir)
					defer index.Close()
					if q == "exit" {
						close(ch)
						return
					}
					query := bleve.NewQueryStringQuery(q)
					searchRequest := bleve.NewSearchRequest(query)
					searchResult, _ := index.Search(searchRequest)
					log.Println(searchResult)
				}()
			}
		}()
		select {
		case <-ch:
		case sig := <-done:
			close(ch)
			log.Printf("[DONE]: 用户输入 '%s' 退出", sig.String())
		}
		log.Println("ByeBye!")
	},
}

var watch bool

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().BoolVarP(&watch, "watch", "w", false, "--watch")
}

func watchChange(done <-chan struct{}) {
	var markM = &markMap{
		RWMutex: sync.RWMutex{},
		m:       map[string]*mark{},
	}
	// 比较差异，新增缺少的文件
	var allMarks = make([]*mark, 0)
	readFile, err := os.ReadFile(jsonFile)
	if err == nil {
		json.NewDecoder(bytes.NewReader(readFile)).Decode(&allMarks)
	}
	for i := range allMarks {
		markM.Add(allMarks[i])
	}
	doChange(markM)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Create {
					log.Println("changed:", event.Name)
					doChange(markM)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	log.Println("watching file: " + systemBookmarkFilePath)
	err = watcher.Add(systemBookmarkFilePath)
	if err != nil {
		log.Fatal(err)
	}
	select {
	case <-done:
		log.Println("watch exit")
		return
	}
}

func doChange(markM *markMap) {
	// 获取用户书签
	file, err := os.ReadFile(systemBookmarkFilePath)
	fatalError(err)
	var data Bookmarks
	fatalError(json.Unmarshal(file, &data))

	realChanged := false
	marks := all(data.Roots.BookmarkBar)
	var qwer = make(map[string]bool)
	markM.Range(func(m *mark) {
		qwer[m.key()] = false
	})
	for _, m := range marks {
		qwer[m.key()] = true
		if !markM.Has(m.key()) {
			log.Println("[NEW]: ", m.Name, m.Url, m.key())
			realChanged = true
			code := 0
			get, err := (&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
				Timeout: 5 * time.Second,
			}).Get(m.Url)
			if err == nil {
				code = get.StatusCode
				if get.StatusCode >= 200 && get.StatusCode < 400 {
					readAll, _ := io.ReadAll(get.Body)
					m.Html = string(readAll)
				} else {
					log.Println(m.Name, get.StatusCode, m.Url)
				}
				get.Body.Close()
			} else {
				log.Println(err)
			}
			markM.Add(&mark{
				Name: m.Name,
				Url:  m.Url,
				Html: m.Html,
				Code: code,
			})
			index, _ := bleve.Open(bleveDir)
			index.Index(fmt.Sprintf("[%s](%s)", m.Name, m.Url), m)
			index.Close()
		}
	}
	for id, b := range qwer {
		if !b {
			log.Println("[DELETE]: ", id)
			realChanged = true
			index, _ := bleve.Open(bleveDir)
			index.Delete(id)
			index.Close()
			markM.Delete(id)
		}
	}

	if realChanged {
		var res []*mark
		markM.Range(func(m *mark) {
			res = append(res, m)
		})
		func() {
			result, _ := json.Marshal(res)
			fatalError(err)
			openFile, _ := os.OpenFile(jsonFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
			defer openFile.Close()
			openFile.Write(result)
			log.Println("写入到文件", jsonFile)
		}()
	}
}

type markMap struct {
	sync.RWMutex
	m map[string]*mark
}

func (mm *markMap) Add(mark *mark) {
	mm.Lock()
	defer mm.Unlock()
	mm.m[mark.key()] = mark
}

func (mm *markMap) Delete(id string) {
	mm.Lock()
	defer mm.Unlock()
	delete(mm.m, id)
}

func (mm *markMap) Has(id string) bool {
	mm.Lock()
	defer mm.Unlock()
	_, ok := mm.m[id]
	return ok
}

func (mm *markMap) Range(fn func(*mark)) {
	mm.Lock()
	defer mm.Unlock()
	for _, m := range mm.m {
		fn(m)
	}
}
