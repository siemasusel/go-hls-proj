package transcoder

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

var command_templ = "ffmpeg -i %s -profile:v baseline -level 3.0 -s 640x360 -start_number 0 -hls_time 10 -hls_list_size 0 -f hls %s"
var allowedExts = map[string]bool{
	".mp4": true,
	".mov": true,
}

type Transcoder struct {
	srcDir     string
	outDir     string
	procVideos []string
	mu         sync.Mutex
}

func New(srcDir string, outDir string) *Transcoder {
	return &Transcoder{srcDir: srcDir, outDir: outDir, procVideos: make([]string, 0, 10)}
}

func (t *Transcoder) Start() {
	go t.startWatcher()
	t.procVideosInDir()
}

func (t *Transcoder) procVideosInDir() {
	err := filepath.Walk(t.srcDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if allowedExts[filepath.Ext(path)] {
			go t.procVideo(path)
		}
		return nil
	})
	if err != nil {
		log.Fatal("ERROR", err)
	}
}

func (t *Transcoder) getOutpath(path string) string {
	filename := filepath.Base(path)
	subdir := strings.ReplaceAll(filename, ".", "_")
	outpath := filepath.Join(t.outDir, subdir, "index.m3u8")
	err := os.MkdirAll(filepath.Dir(outpath), 0755)
	if err != nil {
		log.Fatal(err)
	}
	return outpath
}

func (t *Transcoder) IsProccessing(urlpath string) bool {
	path := path.Join(t.outDir, urlpath)
	for _, p := range t.procVideos {
		if p == path {
			return true
		}
	}
	return false
}

func (t *Transcoder) removeProcVideo(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, p := range t.procVideos {
		if p == path {
			t.procVideos = append(t.procVideos[:i], t.procVideos[i+1:]...)
			break
		}
	}
}

func (t *Transcoder) addProcVideo(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.procVideos = append(t.procVideos, path)
}

func (t *Transcoder) procVideo(path string) {
	filename := filepath.Base(path)
	outpath := t.getOutpath(path)
	command := fmt.Sprintf(command_templ, path, outpath)
	args := strings.Split(command, " ")
	log.Println("Processing file", filename)
	t.addProcVideo(outpath)
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()
	if err != nil {
		log.Fatal("ERROR", err)
	}
	t.removeProcVideo(outpath)
	log.Println("File", filename, "has been successfully processed")
}

func (t *Transcoder) startWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	errCh := make(chan error)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					t.procVideo(event.Name)
				case event.Op&fsnotify.Rename == fsnotify.Rename:
					t.procVideo(event.Name)
				}
			case err := <-watcher.Errors:
				errCh <- err
			}
		}
	}()
	if err := watcher.Add(t.srcDir); err != nil {
		log.Fatal("ERROR", err)
	}
	log.Fatal(<-errCh)
}
