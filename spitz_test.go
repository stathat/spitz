// Copyright (C) 2012 Numerotron Inc.
// Use of this source code is governed by an MIT-style license
// that can be found in the LICENSE file.

package spitz

import (
	//	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func parsedAt(p *Pool, name string) (t time.Time) {
	tmpl, ok := p.tmpls[name]
	if !ok {
		return
	}
	return tmpl.parsedAt
}

func TestRegisterString(t *testing.T) {
	s := New("", false)
	s.RegisterString("test", "<html><head><title>{{ .Title }}</title></head><body>{{ .Body }}</body></html>")
	if s.Len() != 1 {
		t.Errorf("expected 1 template after register, got %d", s.Len())
	}
}

func TestRenderString(t *testing.T) {
	s := New("", false)
	s.RegisterString("test", "<html><head><title>{{ .Title }}</title></head><body>{{ .Body }}</body></html>")
	data := map[string]string{"Title": "the title", "Body": "the body"}
	x, err := s.RenderNoLayoutToString("test", data)
	if err != nil {
		t.Fatal(err)
	}
	if x != "<html><head><title>the title</title></head><body>the body</body></html>" {
		t.Errorf("incorrect render: %q", x)
	}
}

func TestRegister(t *testing.T) {
	dir := createTestDir([]templateFile{
		{"T0.tmpl.html", `T0 template`},
	})
	defer os.RemoveAll(dir)

	s := New(dir, false)
	before := s.Len()
	err := s.Register("T0", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if s.Len()-before != 1 {
		t.Errorf("expected one more template registered, got %d", s.Len()-before)
	}

	x, err := s.RenderNoLayoutToString("T0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if x != "T0 template" {
		t.Errorf("render incorrect: %q", x)
	}
}

func TestRegisterLayout(t *testing.T) {
	dir := createTestDir([]templateFile{
		{"header.tmpl.html", `header template`},
		{"footer.tmpl.html", `footer template`},
	})
	defer os.RemoveAll(dir)

	s := New(dir, false)
	before := len(s.layouts)
	err := s.RegisterLayout("alayout", "header", "footer", "", "")
	if err != nil {
		t.Fatal(err)
	}
	after := len(s.layouts)
	if after-before != 1 {
		t.Errorf("expected one more layout, got %d", after-before)
	}
}

func TestRenderLayout(t *testing.T) {
	dir := createTestDir([]templateFile{
		{"header.tmpl.html", `header template`},
		{"body.tmpl.html", `body template`},
		{"footer.tmpl.html", `footer template`},
	})
	defer os.RemoveAll(dir)
	s := New(dir, false)

	err := s.RegisterLayout("alayout", "header", "footer", "", "")
	if err != nil {
		t.Fatal(err)
	}
	err = s.Register("body", "", "")
	if err != nil {
		t.Fatal(err)
	}
	x, err := s.RenderToString("alayout", "body", nil)
	if err != nil {
		t.Fatal(err)
	}
	if x != "header templatebody templatefooter template" {
		t.Errorf("incorrect render: %q", x)
	}
}

func TestReload(t *testing.T) {
	dir := createTestDir([]templateFile{
		{"T0.tmpl.html", `T0 template`},
	})
	defer os.RemoveAll(dir)
	s := New(dir, false)

	err := s.Register("T0", "", "")
	if err != nil {
		t.Fatal(err)
	}
	x, err := s.RenderNoLayoutToString("T0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if x != "T0 template" {
		t.Errorf("render incorrect: %q", x)
	}

	writeFile(dir, "T0.tmpl.html", "T0 template updated")
	err = s.reloadTemplate("T0")
	if err != nil {
		t.Fatal(err)
	}
	x, err = s.RenderNoLayoutToString("T0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if x != "T0 template updated" {
		t.Errorf("render incorrect: %q", x)
	}
}

func TestAutoReload(t *testing.T) {
	dir := createTestDir([]templateFile{
		{"T0.tmpl.html", `T0 template`},
	})
	defer os.RemoveAll(dir)
	s := New(dir, true)

	err := s.Register("T0", "", "")
	if err != nil {
		t.Fatal(err)
	}
	x, err := s.RenderNoLayoutToString("T0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if x != "T0 template" {
		t.Errorf("render incorrect: %q", x)
	}

	// XXX this sucks, but without this, they get written at the same second and the test fails
	time.Sleep(time.Second)
	writeFile(dir, "T0.tmpl.html", "T0 template updated")
	x, err = s.RenderNoLayoutToString("T0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if x != "T0 template updated" {
		t.Errorf("render incorrect: %q", x)
	}

	// make sure it isn't reparsed every request
	before := parsedAt(s, "T0")
	s.RenderNoLayoutToString("T0", nil)
	after := parsedAt(s, "T0")
	if after != before {
		t.Errorf("template was reparsed even though the file didn't change")
	}
}

func TestAutoReloadOff(t *testing.T) {
	dir := createTestDir([]templateFile{
		{"T0.tmpl.html", `T0 template`},
	})
	defer os.RemoveAll(dir)
	s := New(dir, false)

	err := s.Register("T0", "", "")
	if err != nil {
		t.Fatal(err)
	}
	x, err := s.RenderNoLayoutToString("T0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if x != "T0 template" {
		t.Errorf("render incorrect: %q", x)
	}

	// XXX this sucks, but without this, they get written at the same second and the test fails
	time.Sleep(time.Second)
	writeFile(dir, "T0.tmpl.html", "T0 template updated")
	x, err = s.RenderNoLayoutToString("T0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if x != "T0 template" {
		t.Errorf("render incorrect: %q", x)
	}
}

func BenchmarkRender(b *testing.B) {
	b.StopTimer()
	dir := createTestDir([]templateFile{
		{"header.tmpl.html", `header template`},
		{"body.tmpl.html", `body template`},
		{"footer.tmpl.html", `footer template`},
	})
	defer os.RemoveAll(dir)
	s := New(dir, false)
	err := s.RegisterLayout("alayout", "header", "footer", "", "")
	if err != nil {
		b.Fatal(err)
	}
	err = s.Register("body", "", "")
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		x, err := s.RenderToString("alayout", "body", nil)
		if err != nil {
			b.Fatal(err)
		}
		if x != "header templatebody templatefooter template" {
			b.Errorf("incorrect render: %q", x)
		}
	}
}

/*
func BenchmarkRenderSerial(b *testing.B) {
	b.StopTimer()
	dir := createTestDir([]templateFile{
		{"header.tmpl.html", `header template`},
		{"body.tmpl.html", `body template`},
		{"footer.tmpl.html", `footer template`},
	})
	defer os.RemoveAll(dir)
	ContentDirectory = dir
	err := RegisterLayout("alayout", "header", "footer")
	if err != nil {
		b.Fatal(err)
	}
	err = Register("body")
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var content bytes.Buffer
		err := renderSerial("alayout", "body", nil, &content)
		if err != nil {
			b.Fatal(err)
		}
		if string(content.Bytes()) != "header templatebody templatefooter template" {
			b.Errorf("incorrect render: %q", string(content.Bytes()))
		}
	}
}

// this would probably have a better chance of winning if the templates were more complicated
// (and actually used data)
func BenchmarkRenderParallel(b *testing.B) {
	b.StopTimer()
	dir := createTestDir([]templateFile{
		{"header.tmpl.html", `header template`},
		{"body.tmpl.html", `body template`},
		{"footer.tmpl.html", `footer template`},
	})
	defer os.RemoveAll(dir)
	ContentDirectory = dir
	err := RegisterLayout("alayout", "header", "footer")
	if err != nil {
		b.Fatal(err)
	}
	err = Register("body")
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var content bytes.Buffer
		err := renderParallel("alayout", "body", nil, &content)
		if err != nil {
			b.Fatal(err)
		}
		if string(content.Bytes()) != "header templatebody templatefooter template" {
			b.Errorf("incorrect render: %q", string(content.Bytes()))
		}
	}
}
*/

type page struct {
	Title string
	Body  string
	Count int32
}

func TestRenderStringWithFunc(t *testing.T) {
	s := New("", false)
	s.RegisterString("test", "<html><head><title>{{ .Title }}</title></head><body>{{ .Body }} {{ .Count }} {{ plural \"example\" .Count }}</body></html>")
	data := &page{Title: "the title", Body: "the body", Count: int32(2)}
	x, err := s.RenderNoLayoutToString("test", data)
	if err != nil {
		t.Fatal(err)
	}
	if x != "<html><head><title>the title</title></head><body>the body 2 examples</body></html>" {
		t.Errorf("incorrect render: %q", x)
	}
}

// (from go/src/pkg/text/template/examplefiles_test.go)
// templateFile defines the contents of a template to be stored in a file, for testing.
type templateFile struct {
	name     string
	contents string
}

func createTestDir(files []templateFile) string {
	dir, err := ioutil.TempDir("", "template")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		writeFile(dir, file.name, file.contents)
	}
	return dir
}

func writeFile(dir, name, content string) {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err = io.WriteString(f, content)
	if err != nil {
		log.Fatal(err)
	}
}
