package main

import (
	"github.com/siemasusel/go-hls-proj/app"
)

func main() {
	srv := app.New("0.0.0.0:8000")
	srv.Start()
}
