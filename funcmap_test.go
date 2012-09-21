// Copyright (C) 2012 Numerotron Inc.
// Use of this source code is governed by an MIT-style license
// that can be found in the LICENSE file.

package spitz

import (
	"html/template"
	"testing"
)

func TestSimpleFormat(t *testing.T) {
	input := "hello\nline 2\nline 3\n"
	output := template.HTML("<p>hello</p><p>line 2</p><p>line 3</p>")
	if simpleFormat(input) != output {
		t.Errorf("expected %q, got %q", output, simpleFormat(input))
	}
}

func TestSimpleFormatEscape(t *testing.T) {
	input := "hello <script>alert('message')</script>\nline 2\n"
	output := template.HTML("<p>hello &lt;script&gt;alert(&#39;message&#39;)&lt;/script&gt;</p><p>line 2</p>")
	if simpleFormat(input) != output {
		t.Errorf("expected %q, got %q", output, simpleFormat(input))
	}
}
