package docload

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Fiecher/searchinator"
)

var SupportedExtensions = []string{
	".txt", ".log", ".csv", ".tsv",
	".md", ".markdown",
	".docx",
	".html", ".htm",
	".rtf",
}

func Load(path string) (searchinator.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return searchinator.Document{}, err
	}
	return LoadBytes(filepath.Base(path), data, path)
}

func LoadBytes(name string, data []byte, source string) (searchinator.Document, error) {
	ext := strings.ToLower(filepath.Ext(name))
	text, err := Extract(ext, data)
	if err != nil {
		return searchinator.Document{}, err
	}
	id := strings.TrimSuffix(name, filepath.Ext(name))
	if id == "" {
		id = name
	}
	return searchinator.Document{
		ID:   id,
		Text: strings.TrimSpace(text),
		Meta: map[string]any{"source": source, "format": strings.TrimPrefix(ext, ".")},
	}, nil
}

func Extract(ext string, data []byte) (string, error) {
	switch strings.ToLower(ext) {
	case ".docx":
		return docxText(data)
	case ".html", ".htm":
		return htmlText(data), nil
	case ".rtf":
		return rtfText(data), nil
	default:
		return string(data), nil
	}
}

func docxText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("docload: not a valid docx (zip): %w", err)
	}
	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", errors.New("docload: docx has no word/document.xml")
	}
	rc, err := doc.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	raw, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return html.UnescapeString(splitBlocks(raw)), nil
}

func splitBlocks(raw []byte) string {
	var b strings.Builder
	for _, m := range reDocxToken.FindAllSubmatch(raw, -1) {
		switch {
		case m[1] != nil:
			b.Write(m[1])
		case bytes.HasPrefix(m[0], []byte("<w:p")):
			b.WriteByte('\n')
		case bytes.HasPrefix(m[0], []byte("<w:br")), bytes.HasPrefix(m[0], []byte("<w:cr")):
			b.WriteByte('\n')
		case bytes.HasPrefix(m[0], []byte("<w:tab")):
			b.WriteByte('\t')
		}
	}
	return b.String()
}

var (
	reDocxToken = regexp.MustCompile(`(?s)<w:t(?: [^>]*)?>(.*?)</w:t>|<w:p[ >]|<w:br\b[^>]*/?>|<w:cr\b[^>]*/?>|<w:tab\b[^>]*/?>`)

	reScriptStyle = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	reTag         = regexp.MustCompile(`(?s)<[^>]+>`)
	reWS          = regexp.MustCompile(`[ \t\x{00a0}]{2,}`)

	reRTFControl = regexp.MustCompile(`\\[a-zA-Z]+-?\d* ?|\\[^a-zA-Z]`)
)

func htmlText(data []byte) string {
	s := reScriptStyle.ReplaceAllString(string(data), " ")
	s = reTag.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = reWS.ReplaceAllString(s, " ")
	return s
}

func rtfText(data []byte) string {
	s := reRTFControl.ReplaceAllString(string(data), " ")
	s = strings.NewReplacer("{", " ", "}", " ").Replace(s)
	s = reWS.ReplaceAllString(s, " ")
	return s
}
