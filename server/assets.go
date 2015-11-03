//go:generate go-bindata -pkg main -o assets_files.go assets/...

package main

import (
	"net/http"
	"os"

	"github.com/elazarl/go-bindata-assetfs"
)

// all assets files embedded as a Go library
func FileSystemHandler() http.Handler {
	var handler http.Handler
	if info, err := os.Stat("assets/"); err == nil && info.IsDir() {
		handler = http.FileServer(http.Dir("assets/"))
	} else {
		handler = http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "assets"})
	}
	return handler
}
