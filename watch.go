package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/fsnotify/fsnotify"
)

var markM = &markMap{
	RWMutex: sync.RWMutex{},
	m:       map[string]*mark{},
}

func watchChange(done <-chan struct{}, inited bool) {
	// 比较差异，新增缺少的文件
	if !inited {
		var allMarks = make([]*mark, 0)
		readFile, err := os.ReadFile(jsonFile)
		if err == nil {
			json.NewDecoder(bytes.NewReader(readFile)).Decode(&allMarks)
		}
		for i := range allMarks {
			markM.Add(allMarks[i])
		}
		doChange(markM)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		Error(err)
		os.Exit(1)
	}
	defer watcher.Close()
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Chmod {
					Warnf("\nchanged: %s Op: %s", event.Name, event.Op)
					doChange(markM)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				Error("error:", err)
			}
		}
	}()

	err = watcher.Add(systemBookmarkFilePath)
	if err != nil {
		log.Fatal(err)
	}
	select {
	case <-done:
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
			Infof("[NEW]: name: %s, url: %s", m.Name, m.Url)
			realChanged = true
			code := 0
			get, err := httpClient.Get(m.Url)
			if err == nil {
				code = get.StatusCode
				readAll, _ := io.ReadAll(get.Body)
				if len(readAll) > 0 || (get.StatusCode >= 200 && get.StatusCode < 400) {
					m.Html = string(readAll)
				} else {
					Info(m.Name, get.StatusCode, m.Url)
				}
				get.Body.Close()
			} else {
				Error(err)
			}
			markM.Add(&mark{
				Name: m.Name,
				Url:  m.Url,
				Html: m.Html,
				Code: code,
			})
			index, _ := bleve.Open(bleveDir)
			index.Index(m.key(), m)
			index.Close()
		}
	}
	for id, b := range qwer {
		if !b {
			Errorf("[DELETE]: %v", id)
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
		syncFile(res)
	}
}

func syncFile(res []*mark) {
	result, _ := json.Marshal(res)
	openFile, _ := os.OpenFile(jsonFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	defer openFile.Close()
	openFile.Write(result)
	Infof("\nwrite to %s", jsonFile)
}

type markMap struct {
	sync.RWMutex
	m map[string]*mark
}

func (mm *markMap) Get(id string) *mark {
	mm.RLock()
	defer mm.RUnlock()
	return mm.m[id]
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
