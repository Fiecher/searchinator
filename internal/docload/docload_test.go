package docload

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestExtract_PlainAndMarkdown(t *testing.T) {
	got, err := Extract(".md", []byte("# Title\n\nhello world"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "hello world") {
		t.Errorf("md extract = %q, want it to contain 'hello world'", got)
	}
}

func TestExtract_HTML(t *testing.T) {
	in := `<html><head><style>p{color:red}</style></head><body><h1>Go</h1><p>memory &amp; safety</p><script>x()</script></body></html>`
	got, err := Extract(".html", []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Go", "memory & safety"} {
		if !strings.Contains(got, want) {
			t.Errorf("html extract = %q, want substring %q", got, want)
		}
	}
	if strings.Contains(got, "color:red") || strings.Contains(got, "x()") {
		t.Errorf("html extract leaked script/style: %q", got)
	}
}

func TestExtract_RTF(t *testing.T) {
	in := `{\rtf1\ansi\deff0 {\fonttbl{\f0 Times;}}\f0\fs24 hello \b world\b0 .}`
	got, err := Extract(".rtf", []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") {
		t.Errorf("rtf extract = %q, want 'hello' and 'world'", got)
	}
}

func TestExtract_DOCX(t *testing.T) {
	docXML := `<?xml version="1.0"?>
<w:document xmlns:w="x"><w:body>
<w:p><w:r><w:t>systems programming</w:t></w:r></w:p>
<w:p><w:r><w:t xml:space="preserve">memory </w:t><w:t>safety</w:t></w:r></w:p>
</w:body></w:document>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(docXML)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadBytes("paper.docx", buf.Bytes(), "/tmp/paper.docx")
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID != "paper" {
		t.Errorf("ID = %q, want paper", doc.ID)
	}
	for _, want := range []string{"systems programming", "memory", "safety"} {
		if !strings.Contains(doc.Text, want) {
			t.Errorf("docx text = %q, want substring %q", doc.Text, want)
		}
	}
	if doc.Meta["format"] != "docx" {
		t.Errorf("format meta = %v, want docx", doc.Meta["format"])
	}
}
