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
	"regexp"
	"strings"

	"github.com/taylorskalyo/goreader/epub"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	ErrCmdArgs = 1
	ErrEpub
)

var b struct {
	path          string
	readLaterPath string
	book          *epub.Rootfile
	tocHtml       string
	infoHtml      string
}

func main() {

	// check cmd args
	if len(os.Args) != 2 {
		os.Exit(ErrCmdArgs)
	}
	b.path = os.Args[1] // set .path

	// open book
	rc, err := epub.OpenReader(b.path)
	if err != nil {
		log.Printf("when opening book, %v", err)
		os.Exit(ErrEpub)
	}
	defer rc.Close()
	log.Printf("opened book at %s", b.path)
	b.book = rc.Rootfiles[0] // set .book

	// read later file path
	b.readLaterPath = readLaterPath() // set .readLaterPath

	// build toc
	for _, item := range b.book.Manifest.Items {
		if item.ID == "ncx" {
			b.tocHtml = tocHtml(item) // set .tocHtml
			break
		}
	}
	if b.tocHtml == "" {
		log.Printf("couldn't build table of content")
	} else {
		log.Printf("built table of content")
	}

	b.infoHtml = infoHtml() // set .infoHtml, all set

	if b.readLaterPath != "" {
		http.HandleFunc("/save_brv", saveHandler)
	}

	// create book file request handlers
	for i, item := range b.book.Manifest.Items {
		http.HandleFunc("/"+item.HREF, bookItemHandler(i))
	}

	http.HandleFunc("/", rootHandler)

	// identify the start page
	log.Printf("book ready at http://localhost:8004/")

	// start server on 8004
	log.Fatal(http.ListenAndServe(":8004", nil))
}

func readLaterPath() string {

	// calculate path
	userConfigPath, err := os.UserConfigDir()
	if err != nil {
		log.Printf("when identifying user config dir, %v", err)
		return ""
	}
	hash, err := fileHash(b.path)
	if err != nil {
		log.Printf("when computing book file MD5 hash, %v", err)
		return ""
	}

	dir := filepath.Join(userConfigPath, "brv", "read_later")
	os.MkdirAll(dir, 0755)

	return filepath.Join(dir, fmt.Sprintf("%x", hash))
}

func lastReadJS() string {

	// open rl file
	raw, err := os.ReadFile(b.readLaterPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("when reading read-later file '%s', %v", b.readLaterPath, err)
		}
		return ""
	}

	return string(raw)
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

func infoHtml() string {
	var src string

	appendRow := func(label string, value string) {
		if value != "" {
			src += `<tr><th>` + label + `</th><td>` + html.EscapeString(value) + `</td></tr>`
		}
	}

	md := &b.book.Metadata

	appendRow("Title", md.Title)
	appendRow("Creator", md.Creator)
	appendRow("Contributor", md.Contributor)
	appendRow("Publisher", md.Publisher)
	appendRow("Language", md.Language)
	appendRow("Description", md.Description)
	appendRow("Subject", md.Subject)
	appendRow("Identifier", md.Identifier)
	appendRow("Format", md.Format)
	appendRow("Type", md.Type)
	appendRow("Coverage", md.Coverage)
	appendRow("Relations", md.Relation)
	appendRow("Rights", md.Rights)
	appendRow("Source", md.Source)

	appendRow("Location", b.path)
	appendRow("Read later", b.readLaterPath)

	return src
}

func saveHandler(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("when reading request body, %v", err)
		return
	}

	// write body to savePath
	if err := os.WriteFile(b.readLaterPath, body, 0755); err != nil {
		log.Printf("when saving last read status to '%s', %v", b.readLaterPath, err)
	}
}

func serveBookPage(w http.ResponseWriter, file io.ReadCloser, lastRead string, prev string, next string) {

	if lastRead == "" {
		lastRead = "null"
	}
	js := fmt.Sprintf("const prevHref=\"%s\"; const nextHref=\"%s\"; const lastRead=%s; %s",
		prev, next, lastRead, script)

	// parse file, modify, and write
	tokenizer := html.NewTokenizer(file)
	var err error
	for {
		tokenType := tokenizer.Next()
		token := tokenizer.Token()

		switch tokenType {
		case html.ErrorToken:
			err = tokenizer.Err()
		case html.EndTagToken:
			if token.DataAtom == atom.Body {
				// insert box html
				w.Write([]byte(fmt.Sprintf(appBoxHtmlFmt, b.tocHtml, configFrag, b.infoHtml)))
				// insert script
				w.Write([]byte("<script>" + html.EscapeString(js) + "</script>\n"))
			} else if token.DataAtom == atom.Head {
				// insert style
				w.Write([]byte("<style>" + html.EscapeString(style) + "</style>\n"))
			}
			fallthrough
		default:
			_, err = w.Write([]byte(token.String()))
		}

		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("when tokenizing document, %v", err)
			break
		}
	}

}

func bookItemHandler(i int) func(w http.ResponseWriter, req *http.Request) {
	item := b.book.Manifest.Items[i]

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

		// read-later data, used to recover reading position and customised styles
		var lastRead string
		if b.readLaterPath != "" {
			lastRead = lastReadJS()
			if lastRead != "" {
				// reset position if stored href is not the one requested
				lastHref := lastReadHref(lastRead)
				if !strings.Contains(req.URL.Path, lastHref) {
					lastRead = resetPositionLastRead(lastRead)
				}
			}
		}

		// prev/next page href, used for client navigation
		var prev, next string
		if i-1 >= 0 {
			prev = b.book.Manifest.Items[i-1].HREF
		}
		if i+1 < len(b.book.Manifest.Items) {
			next = b.book.Manifest.Items[i+1].HREF
		}

		serveBookPage(w, file, lastRead, prev, next)

	}
}

func resetPositionLastRead(js string) string {
	re := regexp.MustCompile(`"position" *: *(-?[0-9\.]*)`)
	return re.ReplaceAllLiteralString(js, `"position":0`)
}

func lastReadHref(js string) string {
	re := regexp.MustCompile(`"href" *: *"([^"]*)"`)
	if matches := re.FindStringSubmatch(js); len(matches) > 1 {
		return matches[1]
	} else {
		return ""
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) {

	var startHref string

	if b.readLaterPath != "" {
		startHref = lastReadHref(lastReadJS())
	}
	if startHref == "" {
		startHref = "/" + b.book.Spine.Itemrefs[0].HREF
	}

	w.Header().Add("Location", startHref)
	w.WriteHeader(307)
}

const BoxID = "brv-box"
const ConfigInfoID = "brv-ci"
const InfoID = "brv-info"
const TocID = "brv-toc"

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

//go:embed brv.js
var script string

//go:embed brv.css
var style string

//go:embed config.html
var configFrag string

//go:embed brv.html
var appBoxHtmlFmt string
