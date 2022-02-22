package main

import (
	"errors"
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/highlight/highlighter/ansi"
	"github.com/fatih/color"
	"github.com/peterh/liner"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var topNRegexp = regexp.MustCompile(`-top(\d+)`)
var line = liner.NewLiner()

func init() {
	line.SetCtrlCAborts(true)
}

func search(ch chan struct{}) {
	for {
		err := func() error {
			q, err := line.Prompt("Please enter what you want to search：")
			if err != nil {
				if errors.Is(err, liner.ErrPromptAborted) {
					close(ch)
					return err
				}
				fatalError(err)
				return err
			}

			line.AppendHistory(q)
			q = strings.TrimRight(q, "\n")
			index, err := bleve.Open(bleveDir)
			fatalError(err)
			defer index.Close()
			switch q {
			case "showall":
				markM.Range(func(m *mark) {
					Debugf("key: %s content: %d", m.key(), len(m.Html))
				})
				return nil
			case "exit":
				close(ch)
				return errors.New("exit")
			default:
			}
			size := 10
			hl := false
			if strings.Contains(q, "-h") {
				hl = true
				q = strings.ReplaceAll(q, "-h", "")
			}

			submatch := topNRegexp.FindAllStringSubmatch(q, -1)
			for _, i := range submatch {
				if len(i) == 2 {
					atoi, err := strconv.Atoi(i[1])
					if err == nil && atoi > 0 && atoi < 100 {
						size = atoi
						q = strings.ReplaceAll(q, i[0], "")
					}
				}
			}

			if strings.HasPrefix(q, "show ") {
				id := strings.TrimSpace(strings.TrimLeft(q, "show "))
				get := markM.Get(id)
				if get == nil {
					Warnf("not found %s", id)
					return nil
				}
				Debug(get.Html)
				return nil
			}
			if strings.HasPrefix(q, "refetch ") {
				id := strings.TrimSpace(strings.TrimLeft(q, "refetch "))
				get := markM.Get(id)
				if get == nil {
					Warnf("not found %s", id)
					return nil
				}

				old := get.Html
				get.Html = httpClient.GetResponseString(get.Url)
				Debugf("refetch %s length: %d", get.key(), len(get.Html))
				if old != get.Html {
					markM.Add(get)
					index.Index(get.key(), get)
					var res []*mark
					markM.Range(func(m *mark) {
						res = append(res, m)
					})
					syncFile(res)
				}
				return nil
			}
			query := bleve.NewQueryStringQuery(q)
			searchRequest := bleve.NewSearchRequest(query)
			if highlight || hl {
				searchRequest.Highlight = bleve.NewHighlightWithStyle(ansi.Name)
			}
			searchRequest.Size = size
			searchResult, err := index.Search(searchRequest)
			fatalError(err)
			log.Println(fmtResult(searchResult))
			return nil
		}()
		if err != nil {
			return
		}
	}
}

func fmtResult(sr *bleve.SearchResult) string {
	rv := ""
	if sr.Total > 0 {
		if sr.Request.Size > 0 {
			rv = fmt.Sprintf("%d matches, showing %d through %d, took %s\n", sr.Total, sr.Request.From+1, sr.Request.From+len(sr.Hits), sr.Took)
			for i, hit := range sr.Hits {
				item := markM.Get(hit.ID)
				var length interface{}
				if item != nil {
					length = len(item.Html)
				} else {
					length = "loading..."
				}
				rv += fmt.Sprintf("%3d. %s (%f, content-length: %v)\n", i+sr.Request.From+1, color.GreenString(hit.ID), hit.Score, length)
				for fragmentField, fragments := range hit.Fragments {
					rv += fmt.Sprintf("\t%s\n", fragmentField)
					for _, fragment := range fragments {
						rv += fmt.Sprintf("\t\t%s\n", fragment)
					}
				}
				for otherFieldName, otherFieldValue := range hit.Fields {
					if _, ok := hit.Fragments[otherFieldName]; !ok {
						rv += fmt.Sprintf("\t%s\n", otherFieldName)
						rv += fmt.Sprintf("\t\t%v\n", otherFieldValue)
					}
				}
			}
		} else {
			rv = fmt.Sprintf("%d matches, took %s\n", sr.Total, sr.Took)
		}
	} else {
		rv = "No matches"
	}
	if len(sr.Facets) > 0 {
		rv += fmt.Sprintf("Facets:\n")
		for fn, f := range sr.Facets {
			rv += fmt.Sprintf("%s(%d)\n", fn, f.Total)
			for _, t := range f.Terms.Terms() {
				rv += fmt.Sprintf("\t%s(%d)\n", t.Term, t.Count)
			}
			if f.Other != 0 {
				rv += fmt.Sprintf("\tOther(%d)\n", f.Other)
			}
		}
	}
	return rv
}
