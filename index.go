package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	_ "github.com/duc-cnzj/easy-bookmark/jieba"
	"github.com/schollz/progressbar/v3"
)

func initBookmark(ch chan struct{}) {
	initDone := make(chan struct{}, 1)
	go func() {
		defer func(t time.Time) {
			Infof("init use: %s", time.Since(t))
		}(time.Now())
		// 获取用户书签

		file, err := os.ReadFile(getSystemBookmarkFilePath())
		fatalError(err)
		var data Bookmarks
		fatalError(json.Unmarshal(file, &data))

		// 比较差异，新增缺少的文件
		var allM = make([]*mark, 0)
		readFile, err := os.ReadFile(jsonFile)
		if err == nil {
			json.NewDecoder(bytes.NewReader(readFile)).Decode(&allM)
		}
		var mm = make(map[string]*mark)
		for i := range allM {
			var m = allM[i]
			mm[m.Name] = m
		}
		marks := all(data.Roots.BookmarkBar)
		for i, m := range marks {
			if _, ok := mm[m.Name]; ok {
				marks[i] = mm[m.Name]
			}
		}

		var (
			wg       = &sync.WaitGroup{}
			markChan = make(chan *mark, 1000)
			result   = make([]*mark, 0)
			queue    = make(chan *mark, 100)
		)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for m := range queue {
					func() {
						if m.Html != "" || (m.Code != 429 && m.Code != 0) {
							markChan <- &mark{
								Name: m.Name,
								Url:  m.Url,
								Html: m.Html,
								Code: m.Code,
							}
							return
						}
						var code int
						get, err := httpClient.Get(m.Url)
						if err == nil {
							defer get.Body.Close()
							code = get.StatusCode
							readAll, _ := io.ReadAll(get.Body)
							if len(readAll) > 0 || (get.StatusCode >= 200 && get.StatusCode < 400) {
								m.Html = string(readAll)
							}
						}
						markChan <- &mark{
							Name: m.Name,
							Url:  m.Url,
							Html: m.Html,
							Code: code,
						}
					}()
				}
			}()
		}

		go func() {
			wg.Wait()
			close(markChan)
			//Debug("markChan closed")
		}()
		go func() {
			for i := range marks {
				queue <- marks[i]
			}
			close(queue)
			//Debug("queue closed")
		}()

		Info("creating index file")
		mapping := bleve.NewIndexMapping()

		err = mapping.AddCustomTokenizer("gojieba",
			map[string]interface{}{
				"dictpath":     dictPath,
				"hmmpath":      hmm,
				"userdictpath": userDict,
				"idf":          idf,
				"stop_words":   stopWords,
				"type":         "gojieba",
			},
		)
		fatalError(err)
		err = mapping.AddCustomAnalyzer("gojieba",
			map[string]interface{}{
				"type":      "gojieba",
				"tokenizer": "gojieba",
			},
		)
		fatalError(err)
		mapping.DefaultAnalyzer = "gojieba"

		index, err := bleve.New(bleveDir, mapping)
		fatalError(err)
		bar := progressbar.NewOptions(len(marks),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(30),
		)
		for m := range markChan {
			bar.Add(1)
			index.Index(fmt.Sprintf("[%s](%s)", m.Name, m.Url), m)
			markM.Add(&mark{
				Name: m.Name,
				Url:  m.Url,
				Html: m.Html,
				Code: m.Code,
			})
			result = append(result, m)
		}
		index.Close()
		syncFile(result)
		Info("\nInitialization completed!")
		initDone <- struct{}{}
	}()
	select {
	case <-ch:
	case <-initDone:
	}
}

func fatalError(err error) {
	if err != nil {
		Error(err.Error())
		os.Exit(1)
	}
}

type Item struct {
	Children []Item `json:"children"`
	Type     string `json:"type"`
	Url      string `json:"url"`
	Name     string `json:"name"`
}

type Bookmarks struct {
	Roots struct {
		BookmarkBar Item `json:"bookmark_bar"`
	} `json:"roots"`
}

type mark struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Html string `json:"html"`
	Code int    `json:"code"`
}

func (m mark) key() string {
	return fmt.Sprintf("[%s](%s)", m.Name, m.Url)
}

func all(item Item) []*mark {
	var marks = make([]*mark, 0)
	switch item.Type {
	case "folder":
		if item.Children != nil {
			for _, child := range item.Children {
				marks = append(marks, all(child)...)
			}
		} else {
			return marks
		}
	case "url":
		marks = append(marks, &mark{
			Name: item.Name,
			Url:  item.Url,
		})
	default:
		return nil
	}

	return marks
}
