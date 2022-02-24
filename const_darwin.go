package main

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

var (
	systemBookmarkFilePath = `Library/Application Support/Google/Chrome/Default/Bookmarks`
	bleveDir               = "/tmp/easy-bookmark/cache.bleve"
	jsonFile               = "/tmp/easy-bookmark/bookmark.json"

	dictPath  = "/tmp/easy-bookmark/jieba.dict.utf8"
	hmm       = "/tmp/easy-bookmark/hmm_model.utf8"
	userDict  = "/tmp/easy-bookmark/user.dict.utf8"
	idf       = "/tmp/easy-bookmark/idf.utf8"
	stopWords = "/tmp/easy-bookmark/stop_words.utf8"
)

func getSystemBookmarkFilePath() string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, systemBookmarkFilePath)
}

func InitDict() {
	os.Mkdir("/tmp/easy-bookmark", 0755)

	os.WriteFile(dictPath, getFile(dictPath), 0644)
	os.WriteFile(hmm, getFile(hmm), 0644)
	os.WriteFile(userDict, getFile(userDict), 0644)
	os.WriteFile(idf, getFile(idf), 0644)
	os.WriteFile(stopWords, getFile(stopWords), 0644)
}

func getFile(name string) []byte {
	file, _ := dict.ReadFile("dict/" + filepath.Base(name))
	return file
}
