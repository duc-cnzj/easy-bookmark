package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve"
	_ "github.com/blevesearch/bleve/analysis/analyzer/custom"
	_ "github.com/wangbin/jiebago/tokenizers"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

const systemBookmarkFilePath = `/Users/duc/Library/Application Support/Google/Chrome/Default/Bookmarks`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化书签和索引文件",
	Run: func(cmd *cobra.Command, args []string) {
		initBookmark()
	},
}

func initBookmark() {
	// 获取用户书签
	file, err := os.ReadFile(systemBookmarkFilePath)
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

	wg := &sync.WaitGroup{}
	var (
		result = make([]mark, 0)
		mu     sync.Mutex
	)

	for i := range marks {
		m := marks[i]
		if m.Html != "" || (m.Code != 429 && m.Code != 0) {
			mu.Lock()
			result = append(result, mark{
				Name: m.Name,
				Url:  m.Url,
				Html: m.Html,
				Code: m.Code,
			})
			mu.Unlock()
			continue
		}
		wg.Add(1)
		go func(m *mark) {
			defer wg.Done()
			var code int
			get, err := (&http.Client{}).Get(m.Url)
			if err == nil {
				defer get.Body.Close()
				code = get.StatusCode
				if get.StatusCode >= 200 && get.StatusCode < 400 {
					readAll, _ := io.ReadAll(get.Body)
					m.Html = string(readAll)
				} else {
					log.Println(m.Name, get.StatusCode, m.Url)
				}
			} else {
				log.Println(err)
			}
			mu.Lock()
			result = append(result, mark{
				Name: m.Name,
				Url:  m.Url,
				Html: m.Html,
				Code: code,
			})
			mu.Unlock()
		}(m)
	}
	wg.Wait()

	func() {
		mu.Lock()
		defer mu.Unlock()
		marshal, err := json.Marshal(result)
		fatalError(err)
		openFile, _ := os.OpenFile(jsonFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		defer openFile.Close()
		openFile.Write(marshal)
		log.Println("写入到文件", jsonFile)
	}()

	log.Println("开始创建索引文件")
	mapping := bleve.NewIndexMapping()

	err = mapping.AddCustomTokenizer("jieba",
		map[string]interface{}{
			"file": "jieba.txt",
			"type": "jieba",
		})
	if err != nil {
		panic(err)
	}

	// create a custom analyzer
	err = mapping.AddCustomAnalyzer("jieba",
		map[string]interface{}{
			"type":      "custom",
			"tokenizer": "jieba",
			"token_filters": []string{
				"possessive_en",
				"to_lower",
				"stop_en",
			},
		})

	if err != nil {
		log.Fatal(err)
	}

	mapping.DefaultAnalyzer = "jieba"
	index, err := bleve.New(bleveDir, mapping)
	fatalError(err)
	for _, m := range result {
		index.Index(fmt.Sprintf("[%s](%s)", m.Name, m.Url), m)
	}
	log.Println("初始化完成!")
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func fatalError(err error) {
	if err != nil {
		log.Fatalln(err)
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
