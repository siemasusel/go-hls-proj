package app

import (
	"log"
	"net/http"

	"github.com/siemasusel/go-hls-proj/transcoder"
)

const videosDir = "/var/videos"
const hlsDir = "/var/videos/hls"
const allowOrigin = "*"

type App struct {
	addr       string
	mux        *http.ServeMux
	httpServer *http.Server
	transcoder *transcoder.Transcoder
}

func New(addr string) *App {
	a := &App{
		mux:        http.NewServeMux(),
		addr:       addr,
		transcoder: transcoder.New(videosDir, hlsDir),
	}
	a.initRoutes()
	return a
}

func (a *App) initRoutes() {
	a.mux.Handle("/", a.serveVideoHandler())
}

func (a *App) Start() {
	a.transcoder.Start()
	a.httpServer = &http.Server{Addr: a.addr, Handler: a.mux}
	log.Println("Start listening on", a.addr)
	log.Fatal(a.httpServer.ListenAndServe())
}

func (a *App) serveVideoHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		if a.transcoder.IsProccessing(r.URL.Path) {
			rw.WriteHeader(http.StatusAccepted)
			rw.Write([]byte("Video is still processing, check back later"))
			return
		}
		http.FileServer(http.Dir(hlsDir)).ServeHTTP(rw, r)
	}
}
