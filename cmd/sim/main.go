package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/smackface/wh-lethality/internal/profiles"
	"github.com/smackface/wh-lethality/internal/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	data := flag.String("data", "data/units", "Path to unit profile JSON directory")
	tmplDir := flag.String("templates", "web/templates", "Path to HTML templates directory")
	staticDir := flag.String("static", "web/static", "Path to static assets directory")
	flag.Parse()

	store := profiles.NewStore(*data)

	tmplFS := os.DirFS(*tmplDir)
	statFS := os.DirFS(*staticDir)

	srv, err := web.New(store, tmplFS, statFS)
	if err != nil {
		log.Fatalf("web.New: %v", err)
	}

	fmt.Printf("wh-lethality running at http://localhost%s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, srv))
}
