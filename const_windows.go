package main

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

var (
	tmpDir = os.TempDir()
	systemBookmarkFilePath = `\AppData\Local\Google\Chrome\User Data\Default\Bookmarks`
	bleveDir               = filepath.Join(tmpDir + "easy-bookmark", "cache.bleve")
	jsonFile               = filepath.Join(tmpDir + "easy-bookmark", "bookmark.json")

	dictPath  = filepath.Join(tmpDir + "easy-bookmark", "jieba.dict.utf8")
	hmm       = filepath.Join(tmpDir + "easy-bookmark", "hmm_model.utf8")
	userDict  = filepath.Join(tmpDir + "easy-bookmark", "user.dict.utf8")
	idf       = filepath.Join(tmpDir + "easy-bookmark", "idf.utf8")
	stopWords = filepath.Join(tmpDir + "easy-bookmark", "stop_words.utf8")
)

func getSystemBookmarkFilePath() string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, systemBookmarkFilePath)
}

func InitDict() {
	os.Mkdir(filepath.Join(tmpDir, "easy-bookmark"), 0755)

	os.WriteFile(dictPath, getFile(dictPath), 0644)
	os.WriteFile(hmm, getFile(hmm), 0644)
	os.WriteFile(userDict, getFile(userDict), 0644)
	os.WriteFile(idf, getFile(idf), 0644)
	os.WriteFile(stopWords, getFile(stopWords), 0644)
}

func getFile(name string) []byte {
	file, _ := dict.ReadFile(filepath.Join("dict", filepath.Base(name)))
	return file
}
