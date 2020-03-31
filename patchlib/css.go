package patchlib

import (
	"io"

	"github.com/riking/cssparse/tokenizer"
)

// IsCSS uses a heuristic to check if the input is likely to be CSS. There may
// be false negatives for invalid CSS or for CSS documents with more rules than
// properties, and false positives for CSS-like languages. An error is returned
// if and only if the io.Reader returns an error.
func IsCSS(r io.Reader) (bool, error) {
	tk := tokenizer.NewTokenizer(r)
	var seenNonWhitespace bool
	var openBraceCount, closeBraceCount, propCount int
	var prev, cur tokenizer.Token
	for {
		t := tk.Next()
		if err := tk.Err(); err != nil {
			if err == io.EOF {
				break
			}
			return false, tk.Err()
		}
		prev, cur = cur, t
		if !seenNonWhitespace {
			if cur.Type != tokenizer.TokenS {
				if cur.Type != tokenizer.TokenIdent && cur.Type != tokenizer.TokenColon && cur.Type != tokenizer.TokenAtKeyword && !(cur.Type == tokenizer.TokenDelim && (cur.Value == "*" || cur.Value == ".")) && cur.Type != tokenizer.TokenHash && cur.Type != tokenizer.TokenOpenBracket {
					return false, nil // doesn't start with an identifier (for a selector)
				}
				seenNonWhitespace = true
			}
			continue
		}
		switch cur.Type {
		case tokenizer.TokenOpenBrace:
			openBraceCount++
		case tokenizer.TokenCloseBrace:
			closeBraceCount++
		case tokenizer.TokenColon:
			if prev.Type == tokenizer.TokenIdent && openBraceCount > closeBraceCount {
				propCount++ // is likely a property if it has an identifier before and is inside an unclosed block (i.e. not a selector)
			}
		}
	}
	if openBraceCount != closeBraceCount {
		return false, nil // different number of open/close braces
	}
	if openBraceCount == 0 {
		return false, nil // no rules
	}
	if openBraceCount > propCount {
		return false, nil // more blocks than properties
	}
	return true, nil
}
