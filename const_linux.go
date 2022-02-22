package main

import (
	"github.com/mitchellh/go-homedir"
	"path/filepath"
	"os"
)

const (
	systemBookmarkFilePath = `.config/google-chrome/Default/Bookmarks`
	bleveDir = "/tmp/cache.bleve"
	jsonFile = "/tmp/bookmark.json"
)

func getSystemBookmarkFilePath() string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, systemBookmarkFilePath)
}

func InitDict()  {
	os.Mkdir("/tmp/easy-bookmark", 0755)

	os.WriteFile(dictPath, getFile(dictPath), 0644)
	os.WriteFile(hmm, getFile(hmm), 0644)
	os.WriteFile(userDict, getFile(userDict), 0644)
	os.WriteFile(idf, getFile(idf), 0644)
	os.WriteFile(stopWords, getFile(stopWords), 0644)
}

func getFile(name string) []byte  {
	file, _ := dict.ReadFile(filepath.Join("dict", filepath.Base(name)))
	return file
}