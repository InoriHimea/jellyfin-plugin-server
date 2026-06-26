package handler

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

//go:embed all:dist
var distFS embed.FS

var uiHandler fasthttp.RequestHandler

func init() {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	// SPA fallback: always serve index.html for unknown paths
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := sub.Open(r.URL.Path[1:])
		if err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
	uiHandler = fasthttpadaptor.NewFastHTTPHandler(spaHandler)
}

func ServeUI(ctx *fasthttp.RequestCtx) {
	uiHandler(ctx)
}
