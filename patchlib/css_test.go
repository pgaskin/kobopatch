package patchlib

import (
	"strings"
	"testing"
)

func TestIsCSS(t *testing.T) {
	for _, tc := range []struct {
		in  string
		out bool
		why string
	}{
		{"", false, "no rules are present"},
		{"asd", false, "no rules are present"},
		{" asd", false, "no rules are present"},
		{"asd {", false, "there are incomplete rules"},
		{"asd {}", false, "there aren't any properties (technically it is CSS, but for all intents and purposes it isn't)"},
		{"asd {asd:fgh}", true, "it obviously is"},
		{" asd {asd: fgh}", true, "it obviously is"},
		{" asd {asd: fgh}/*}*/", true, "it obviously is"},
		{"    * {asd: fgh}/*}*/", true, "it obviously is"},
		{" asd {asd: url(:sdf{}dfg)}", true, "it obviously is"},
		{" asd {asd: fgh; dfg: asd}", true, "it obviously is"},
		{" @media print { asd {asd: fgh; dfg: asd} }", true, "it obviously is"},
		{"[prop]{dfg: asd}", true, "it obviously is"},
		{".class{dfg: asd}", true, "it obviously is"},
		{"#id{dfg: asd}", true, "it obviously is"},
		{":hover{dfg: asd}", true, "it obviously is"},
		{":hover{dfg: asd} fgh {content: 'dfg}'}", true, "it obviously is"},
		{":hover{dfg: asd; fgh: asd;} fgh {dfgdfgdfg}", true, "it's syntactically valid"},
		{"function asd() { fgh(); }", false, "there's more blocks than rules (i.e. it's JavaScript)"},
		{"<html><span style='asd: dfg'>dfgdfg</span></html>", false, "it obviously isn't (i.e. it's HTML)"},
		{"<html><span style='asd: dfg'>dfgdfg</span><style>dfg {asd: dfg} fgh {dfg: ert; fgh: asd}</style></html>", false, "it obviously isn't (i.e. it's HTML, even though it might contain CSS in it), as it doesn't start with a selector"},
	} {
		ic, err := IsCSS(strings.NewReader(tc.in))
		if err != nil {
			panic(err)
		}
		var st string
		if !tc.out {
			st = "not "
		}
		if ic != tc.out {
			t.Errorf("incorrect: %#v should %sbe CSS because %s, but IsCSS()=%t", tc.in, st, tc.why, ic)
		} else {
			t.Logf("correct: %#v is %sCSS because %s, so IsCSS()=%t", tc.in, st, tc.why, ic)
		}
	}
}
