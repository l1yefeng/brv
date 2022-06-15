package main

import (
	_ "embed"
	"github.com/taylorskalyo/goreader/epub"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	ErrCmdArgs = 1
	ErrEpub
)

const TocID = "brv-toc"

func main() {
	// check cmd args
	if len(os.Args) != 2 {
		os.Exit(ErrCmdArgs)
	}

	bookPath := os.Args[1]
	rc, err := epub.OpenReader(bookPath)
	if err != nil {
		log.Printf("when opening book, %v", err)
		os.Exit(ErrEpub)
	}
	defer rc.Close()
	log.Printf("opened book at %s", bookPath)

	book := rc.Rootfiles[0]

	// build toc
	var tocHtml string
	for _, item := range book.Manifest.Items {
		if item.ID == "ncx" {
			tocHtml = makeTocHtml(item)
			break
		}
	}

	if tocHtml == "" {
		log.Printf("couldn't build table of content")
	} else {
		log.Printf("built table of content")
	}

	// create handlers
	for _, item := range book.Manifest.Items {
		http.HandleFunc("/"+item.HREF, makeHandler(item, tocHtml))
	}

	// identify the start page
	startPage := book.Spine.Itemrefs[0]
	log.Printf("book ready at http://localhost:8004/%s", startPage.HREF)

	// start server on 8004
	log.Fatal(http.ListenAndServe(":8004", nil))
}

func makeTocHtml(ncx epub.Item) string {
	file, err := ncx.Open()
	defer func() {
		if err = file.Close(); err != nil {
			log.Printf("when closing '%s', %v", dumpItem(ncx), err)
		}
	}()

	if err != nil {
		log.Printf("when opening item '%s', %v", dumpItem(ncx), err)
		return ""
	}

	var src string
	var tagStack []string
	var pendingText string

	tokenizer := html.NewTokenizer(file)
	for {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		const phantom = ""

		switch tokenType {

		case html.StartTagToken:

			switch token.Data {
			case "doctitle": // assuming it appears before toc
				src += "<h1>"
			case "navmap":
				src += "<menu>"
			case "navpoint":
				if tagStack[len(tagStack)-1] == "navpoint" {
					tagStack = append(tagStack, phantom)
					src += "<menu>"
				}
				src += "<li>"
			}

			tagStack = append(tagStack, token.Data)

		case html.EndTagToken:

			switch token.Data {
			case "doctitle":
				src += pendingText + "</h1>"
			case "navmap":
				src += "</menu>"
			case "navpoint":
				for tagStack[len(tagStack)-1] == phantom {
					src += "</menu>"
					tagStack = tagStack[:len(tagStack)-1]
				}
				src += "</li>"
			}

			tagStack = tagStack[:len(tagStack)-1]

		case html.SelfClosingTagToken:
			if token.Data == "content" { // assuming right after navLabel
				var href string
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						href = attr.Val
						break
					}
				}
				src += `<a href="/` + href + `">` + pendingText + "</a>"
			}
		case html.TextToken:
			if len(tagStack) > 0 && tagStack[len(tagStack)-1] == "text" {
				pendingText = html.EscapeString(token.Data)
			}
		case html.CommentToken:
		case html.DoctypeToken:
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return src
			} else {
				log.Printf("when tokenizing '%s', %v", dumpItem(ncx), tokenizer.Err())
				return ""
			}
		}
	}
}

func makeHandler(item epub.Item, toc string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {

		file, err := item.Open()
		defer func() {
			if err = file.Close(); err != nil {
				log.Printf("when closing '%s', %v", dumpItem(item), err)
			}
		}()

		if err != nil {
			log.Printf("when opening item '%s', %v", dumpItem(item), err)
			return
		}

		w.Header().Set("Content-Type", item.MediaType)

		// respond immediately unless is document
		if !strings.Contains(item.MediaType, "html") {
			buf := make([]byte, 1024)
			for {
				n, err := file.Read(buf)
				if err != nil && err != io.EOF {
					log.Printf("when reading item '%s', %v", dumpItem(item), err)
					return
				}
				if n == 0 {
					return
				}
				if _, err := w.Write(buf[:n]); err != nil {
					log.Printf("when writing to respond, %v", err)
					return
				}
			}
		}

		// parse file, modify, and respond
		tokenizer := html.NewTokenizer(file)
		for {
			tokenType := tokenizer.Next()
			token := tokenizer.Token()

			switch tokenType {
			case html.ErrorToken:
				err = tokenizer.Err()
			case html.EndTagToken:
				if token.DataAtom == atom.Body {
					// insert toc node
					_, err = w.Write([]byte(`<div id="` + TocID + `" style="display:none"><aside>` + toc + "</aside></div>\n"))
					// insert script
					_, err = w.Write([]byte("<script>" + script + "</script>\n"))
				} else if token.DataAtom == atom.Head {
					// insert style
					_, err = w.Write([]byte("<style>" + style + "</style>\n"))
				}
				fallthrough
			default:
				_, err = w.Write([]byte(token.String()))
			}

			if err == io.EOF {
				break
			} else if err != nil {
				log.Printf("when tokenizing document '%s', %v", dumpItem(item), err)
				break
			}
		}
	}
}

func dumpItem(item epub.Item) string {
	return item.ID + " <" + item.HREF + ">"
}

//go:embed brv.js
var script string

//go:embed brv.css
var style string
