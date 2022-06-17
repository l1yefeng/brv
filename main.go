package main

import (
	"crypto/md5"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/taylorskalyo/goreader/epub"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	ErrCmdArgs = 1
	ErrEpub
)

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
	var toc string
	for _, item := range book.Manifest.Items {
		if item.ID == "ncx" {
			toc = tocHtml(item)
			break
		}
	}

	if toc == "" {
		log.Printf("couldn't build table of content")
	} else {
		log.Printf("built table of content")
	}

	var lastRead string
	if bookRLPath := readLaterPath(bookPath); bookRLPath != "" {
		lastRead = lastReadJS(bookRLPath)
		http.HandleFunc("/save_brv", saveHandler(bookRLPath))
	} else {
		lastRead = emptyLastRead
	}

	// create book file request handlers
	for _, item := range book.Manifest.Items {
		http.HandleFunc("/"+item.HREF, bookItemHandler(item, toc, metadataHtml(book.Metadata), lastRead))
	}

	// identify the start page
	startPage := book.Spine.Itemrefs[0]
	log.Printf("book ready at http://localhost:8004/%s", startPage.HREF)

	// start server on 8004
	log.Fatal(http.ListenAndServe(":8004", nil))
}

func readLaterPath(bookPath string) string {

	// calculate path
	userConfigPath, err := os.UserConfigDir()
	if err != nil {
		log.Printf("when identifying user config dir, %v", err)
		return ""
	}
	hash, err := fileHash(bookPath)
	if err != nil {
		log.Printf("when computing book file MD5 hash, %v", err)
		return ""
	}

	dir := filepath.Join(userConfigPath, "brv", "read_later")
	os.MkdirAll(dir, 0755)

	return filepath.Join(dir, fmt.Sprintf("%x", hash))
}

func lastReadJS(path string) string {

	// open rl file
	raw, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("when reading read-later file '%s', %v", path, err)
		}
		return emptyLastRead
	}

	return "const lastRead = " + string(raw) + ";"
}

func tocHtml(ncx epub.Item) string {
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

func metadataHtml(md epub.Metadata) string {
	src := "<table>"

	appendRow := func(label string, value string) {
		if value != "" {
			src += `<tr><th>` + label + `</th><td>` + html.EscapeString(value) + `</td></tr>`
		}
	}
	appendRow("Title", md.Title)
	appendRow("Creator", md.Creator)
	appendRow("Contributor", md.Contributor)
	appendRow("Publisher", md.Publisher)
	appendRow("Language", md.Language)
	appendRow("Identifier", md.Identifier)
	appendRow("Description", md.Description)

	src += "</table>"
	return src
}

func saveHandler(savePath string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.Printf("when reading request body, %v", err)
			return
		}

		// write body to savePath
		if err := os.WriteFile(savePath, body, 0755); err != nil {
			log.Printf("when saving last read status to '%s', %v", savePath, err)
		}
	}
}

func bookItemHandler(item epub.Item, toc string, metadata string, lastRead string) func(w http.ResponseWriter, req *http.Request) {
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
					// insert box html
					_, err = w.Write([]byte(appBoxHtml(toc, metadata)))
					// insert script
					_, err = w.Write([]byte("<script>" + html.EscapeString(lastRead+script) + "</script>\n"))
				} else if token.DataAtom == atom.Head {
					// insert style
					_, err = w.Write([]byte("<style>" + html.EscapeString(style) + "</style>\n"))
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

const BoxID = "brv-box"
const ConfigInfoID = "brv-ci"
const InfoID = "brv-info"
const TocID = "brv-toc"

func appBoxHtml(toc string, metadata string) string {
	src := `<div id="` + BoxID + `" style="display:none">`
	src += `<aside id="` + TocID + `">` + toc + "</aside>"
	infoFrag := `<section id="` + InfoID + `"><h2>Book information</h2>` + metadata + "</section>"
	src += `<aside id="` + ConfigInfoID + `">` + configFrag + infoFrag + aboutFrag + "</aside>"
	src += "</div>"
	return src
}

func fileHash(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

func dumpItem(item epub.Item) string {
	return item.ID + " <" + item.HREF + ">"
}

const emptyLastRead = "const lastRead = null;"

//go:embed brv.js
var script string

//go:embed brv.css
var style string

//go:embed config.html
var configFrag string

//go:embed about.html
var aboutFrag string
