package parser

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxParagraphRunes = 65535

var ErrEmpty = errors.New("parser: empty content")

type Chunk struct {
	Source  string
	Index   int
	Total   int
	Content string
}

var (
	paragraphRe       = regexp.MustCompile(`\r?\n[\t ]*(?:\r?\n)+`)
	sentenceTerminals = map[rune]struct{}{
		'.': {}, '?': {}, '!': {},
		'。': {}, '？': {}, '！': {},
	}
)

func makeChunks(ctx context.Context, source string, contents []string) ([]Chunk, error) {
	var pieces []string
	for _, c := range contents {
		t := strings.TrimSpace(c)
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) <= maxParagraphRunes {
			pieces = append(pieces, t)
			continue
		}
		pieces = append(pieces, splitBySentence(t, maxParagraphRunes)...)
	}
	if len(pieces) == 0 {
		return nil, nil
	}

	total := len(pieces)
	docs := make([]Chunk, 0, total)
	for i, p := range pieces {
		if err := ctx.Err(); err != nil {
			return docs, err
		}
		docs = append(docs, Chunk{
			Source:  source,
			Index:   i + 1,
			Total:   total,
			Content: p,
		})
	}
	return docs, nil
}

func splitParagraphs(ctx context.Context, source, text string) ([]Chunk, error) {
	return makeChunks(ctx, source, paragraphRe.Split(text, -1))
}

func splitBySentence(text string, max int) []string {
	runes := []rune(text)
	if len(runes) <= max {
		if seg := strings.TrimSpace(string(runes)); seg != "" {
			return []string{seg}
		}
		return nil
	}

	var boundaries []int
	for i, r := range runes {
		if _, ok := sentenceTerminals[r]; ok {
			boundaries = append(boundaries, i)
		}
	}

	var chunks []string
	start := 0
	bIdx := 0
	for start < len(runes) {
		if len(runes)-start <= max {
			if seg := strings.TrimSpace(string(runes[start:])); seg != "" {
				chunks = append(chunks, seg)
			}
			break
		}
		limit := start + max - 1
		for bIdx < len(boundaries) && boundaries[bIdx] < start {
			bIdx++
		}
		cut := -1
		for bIdx < len(boundaries) && boundaries[bIdx] <= limit {
			cut = boundaries[bIdx]
			bIdx++
		}
		if cut < start {
			cut = limit
		}
		if seg := strings.TrimSpace(string(runes[start : cut+1])); seg != "" {
			chunks = append(chunks, seg)
		}
		start = cut + 1
	}
	return chunks
}
