package main

import (
	"github.com/mitchellh/go-homedir"
	"path/filepath"
)

var (
	systemBookmarkFilePath = `Library/Application Support/Google/Chrome/Default/Bookmarks`
	bleveDir = "/tmp/cache.bleve"
	jsonFile = "/tmp/bookmark.json"
)

func getSystemBookmarkFilePath() string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, systemBookmarkFilePath)
}
