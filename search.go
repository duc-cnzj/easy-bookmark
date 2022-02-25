package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/highlight/highlighter/ansi"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

var topNRegexp = regexp.MustCompile(`-top(\d+)`)

//go:embed logo.txt
var logo string

func search(ch chan struct{}) {
	color.Cyan(logo)
	showHelp()
	for {
		err := func() error {
			fmt.Println("search: ")
			fmt.Printf("=> ")
			q, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			q = strings.TrimRight(q, "\n")
			if q == "" {
				showHelp()
				return nil
			}
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
			if strings.Contains(q, " -h") {
				hl = true
				q = strings.ReplaceAll(q, " -h", "")
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
				atoi, err := strconv.Atoi(id)
				if err != nil {
					return nil
				}
				for i, s := range lastResult {
					if i == atoi-1 {
						get := markM.Get(s)
						if get == nil {
							Warnf("not found %s", s)
							return nil
						}
						Debug(get.Html)
						Info(get.key())
						break
					}
				}

				return nil
			}
			if strings.HasPrefix(q, "refetch ") {
				id := strings.TrimSpace(strings.TrimLeft(q, "refetch "))
				atoi, err := strconv.Atoi(id)
				if err != nil {
					return nil
				}
				for i, s := range lastResult {
					if i == atoi-1 {
						get := markM.Get(s)
						if get == nil {
							Warnf("not found %s", s)
							return nil
						}
						old := get.Html
						get.Html = httpClient.GetResponseString(get.Url)
						Debugf("refetch %s length: %d %d", get.key(), len(get.Html), len(old))
						if old != get.Html {
							markM.Add(get)
							index.Index(get.key(), get)
							var res []*mark
							markM.Range(func(m *mark) {
								res = append(res, m)
							})
							syncFile(res)
						}
						break
					}
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

func showHelp() {
	color.Cyan(`Command:
  showall:
        show all bookmarks info. eg: "showall"
  show $id:
        show one bookmark. eg: "show 1"
  refetch $id:
        refetch bookmark html content. eg: "refetch 1"
  -h:
        show search hits. eg: "book -h"
  -topN:
        limit results. eg: "book -top1"
`)
}

var lastResult []string

func fmtResult(sr *bleve.SearchResult) string {
	lastResult = nil
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
				lastResult = append(lastResult, hit.ID)
				if len(hit.Fragments) > 0 {
					bf := &bytes.Buffer{}

					table := tablewriter.NewWriter(bf)
					table.SetHeader([]string{"#", "type", "matched data"})
					table.SetColMinWidth(0, 2)
					table.SetColMinWidth(1, 6)
					table.SetColMinWidth(2, 100)
					table.SetRowLine(true)
					i := 0
					for fragmentField, fragments := range hit.Fragments {
						for _, fragment := range fragments {
							i++
							table.Append([]string{strconv.Itoa(i), fragmentField, fragment})
						}
					}

					table.Render()
					rv += fmt.Sprintf("%s", bf.String())
				}
			}
		} else {
			rv = fmt.Sprintf("%d matches, took %s\n", sr.Total, sr.Took)
		}
	} else {
		rv = "No matches"
	}
	return rv
}
