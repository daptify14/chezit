package tui

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

const defaultChromaStyleName = "catppuccin-mocha"

// highlightCode applies syntax highlighting to source code and returns ANSI-colored text.
// It detects the language from the filename. If detection fails or highlighting errors,
// it returns the original source unchanged.
func highlightCode(source, filename string) string {
	lexer := detectLexer(filename)
	if lexer == nil {
		return source
	}
	lexer = chroma.Coalesce(lexer)

	styleName := activeTheme.ChromaStyleName
	if styleName == "" {
		styleName = defaultChromaStyleName
	}
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return source
	}

	result := buf.String()
	// Chroma may add a trailing newline; trim to match original
	if !strings.HasSuffix(source, "\n") {
		result = strings.TrimRight(result, "\n")
	}
	return result
}

// detectLexer finds the appropriate chroma lexer for the given filename.
// For .tmpl files, prefers the inner extension (e.g. "config.yaml.tmpl" → YAML)
// over chroma's native .tmpl match (Cheetah).
func detectLexer(filename string) chroma.Lexer {
	if filename == "" {
		return nil
	}

	name := filepath.Base(filename)
	if name == "" || name == "." {
		return nil
	}

	// Try inner extension first for .tmpl files.
	if inner, ok := trimTemplateSuffix(name); ok {
		if lexer := lexers.Match(inner); lexer != nil {
			return lexer
		}
	}

	return lexers.Match(name)
}

func trimTemplateSuffix(name string) (string, bool) {
	const suffix = ".tmpl"
	if len(name) <= len(suffix) {
		return "", false
	}
	if !strings.EqualFold(name[len(name)-len(suffix):], suffix) {
		return "", false
	}
	return name[:len(name)-len(suffix)], true
}
