package spitz

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path"
	"time"
)

var Verbose bool

type tfile struct {
	filename string
	modTime  time.Time
}

type tfiles []*tfile

func singleFile(filename string) tfiles {
	return tfiles{&tfile{filename: filename}}
}

func multiFile(filenames []string) tfiles {
	tf := make(tfiles, len(filenames))
	for i, v := range filenames {
		tf[i] = &tfile{filename: v}
	}
	return tf
}

func (tf tfiles) name() string {
	if len(tf) == 0 {
		return ""
	}
	return path.Base(tf[0].filename)
}

func (tf tfiles) filenames() []string {
	names := make([]string, len(tf))
	for i, v := range tf {
		names[i] = v.filename
	}
	return names
}

func (tf tfiles) setModTimes() error {
	for _, v := range tf {
		fi, err := os.Stat(v.filename)
		if err != nil {
			return err
		}
		v.modTime = fi.ModTime()
	}
	return nil
}

func (tf tfiles) modifiedSince(t time.Time) bool {
	for _, v := range tf {
		fi, err := os.Stat(v.filename)
		if err == nil && fi.ModTime().After(t) {
			return true
		}
	}
	return false
}

type tmpl struct {
	name       string
	files      tfiles
	plate      *template.Template
	parsedAt   time.Time
	leftDelim  string
	rightDelim string
}

type layout struct {
	header string
	footer string
}

type Pool struct {
	contentDir string
	reload     bool
	tmpls      map[string]*tmpl
	layouts    map[string]*layout
}

func New(contentDir string, reload bool) *Pool {
	pool := &Pool{contentDir: contentDir, reload: reload}
	pool.tmpls = make(map[string]*tmpl)
	pool.layouts = make(map[string]*layout)
	return pool
}

// Register adds the template at ContentDirectory + name + .tmpl.html to the set
// of registered templates.
func (p *Pool) Register(name, leftDelim, rightDelim string) error {
	ename := fmt.Sprintf("%s.tmpl.html", name)
	filename := path.Join(p.contentDir, ename)
	nt := &tmpl{name: name, files: singleFile(filename), leftDelim: leftDelim, rightDelim: rightDelim}
	err := nt.parse()
	if err != nil {
		return err
	}
	p.tmpls[name] = nt
	return nil
}

func (p *Pool) RegisterLayout(name, header, footer string, leftDelim, rightDelim string) error {
	err := p.Register(header, leftDelim, rightDelim)
	if err != nil {
		return err
	}
	err = p.Register(footer, leftDelim, rightDelim)
	if err != nil {
		return err
	}
	p.layouts[name] = &layout{header, footer}
	return nil
}

func (p *Pool) RegisterString(name, content string) error {
	t, err := template.New(name).Funcs(funcMap).Parse(content)
	if err != nil {
		return err
	}
	p.tmpls[name] = &tmpl{plate: t}
	return nil
}

func (p *Pool) RegisterMulti(name, leftDelim, rightDelim string, filenames ...string) error {
	paths := make([]string, len(filenames))
	for i, v := range filenames {
		paths[i] = path.Join(p.contentDir, fmt.Sprintf("%s.tmpl.html", v))
	}
	files := multiFile(paths)

	nt := &tmpl{name: name, files: files, leftDelim: leftDelim, rightDelim: rightDelim}
	err := nt.parse()
	if err != nil {
		return err
	}
	p.tmpls[name] = nt
	return nil
}

func (p *Pool) Render(layout, name string, data interface{}, wr io.Writer) error {
	lout, ok := p.layouts[layout]
	if !ok {
		return errors.New(fmt.Sprintf("layout %q not found", layout))
	}

	err := p.templateRun(lout.header, data, wr)
	if err != nil {
		return err
	}
	err = p.templateRun(name, data, wr)
	if err != nil {
		return err
	}
	err = p.templateRun(lout.footer, data, wr)
	return err
}

func (p *Pool) RenderMulti(layout, name, innerName string, data interface{}, wr io.Writer) error {

	lout, ok := p.layouts[layout]
	if !ok {
		return errors.New(fmt.Sprintf("layout %q not found", layout))
	}

	err := p.templateRun(lout.header, data, wr)
	if err != nil {
		return err
	}

	err = p.templateRunInner(name, innerName, data, wr)
	if err != nil {
		return err
	}

	err = p.templateRun(lout.footer, data, wr)
	return err
}

func (p *Pool) RenderNoLayout(name string, data interface{}, wr io.Writer) error {
	return p.templateRun(name, data, wr)
}

func (p *Pool) RenderToString(layout, template string, data interface{}) (string, error) {
	var b bytes.Buffer
	err := p.Render(layout, template, data, &b)
	if err != nil {
		return "", err
	}
	return string(b.Bytes()), nil
}

func (p *Pool) RenderNoLayoutToString(template string, data interface{}) (string, error) {
	var b bytes.Buffer
	err := p.RenderNoLayout(template, data, &b)
	if err != nil {
		return "", err
	}
	return string(b.Bytes()), nil
}

func (p *Pool) Len() int {
	return len(p.tmpls)
}

func (p *Pool) templateRun(name string, data interface{}, wr io.Writer) error {
	t, ok := p.tmpls[name]
	if !ok {
		return errors.New(fmt.Sprintf("template %q not found", name))
	}
	if p.reload {
		t.checkReload()
	}
	return t.plate.Execute(wr, data)
}

func (p *Pool) templateRunInner(name, innerName string, data interface{}, wr io.Writer) error {
	t, ok := p.tmpls[name]
	if !ok {
		return errors.New(fmt.Sprintf("template %q not found", name))
	}
	if p.reload {
		t.checkReload()
	}
	return t.plate.ExecuteTemplate(wr, innerName, data)
}

func (t *tmpl) parse() error {
	if len(t.files) == 0 {
		return errors.New(fmt.Sprintf("no files for template %q", t.name))
	}
	tm, err := template.New(t.files.name()).Delims(t.leftDelim, t.rightDelim).Funcs(funcMap).ParseFiles(t.files.filenames()...)
	if err != nil {
		return err
	}
	t.plate = tm
	t.parsedAt = time.Now()
	return t.files.setModTimes()
}

func (t *tmpl) checkReload() {
	if len(t.files) == 0 {
		if Verbose {
			log.Printf("no files")
		}
		return
	}
	if t.files.modifiedSince(t.parsedAt) {
		if Verbose {
			log.Printf("template %s changed.  reparsing...", t.name)
		}
		t.parse()
	}
}

func (p *Pool) reloadTemplate(name string) error {
	t, ok := p.tmpls[name]
	if !ok {
		return errors.New(fmt.Sprintf("template %q not registered", name))
	}
	return t.parse()
}
