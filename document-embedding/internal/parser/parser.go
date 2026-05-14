package parser

import (
	"bytes"
	"errors"
	"strings"
	"unicode"
)

type ParsedDocument struct {
	FileName string
	Chunks   []Chunk
}

type Chunk struct {
	Content string
	Page    int
}

type DocumentParser interface {
	Parse(data []byte, fileName string) (*ParsedDocument, error)
}

type SimpleParser struct{}

func NewSimpleParser() *SimpleParser {
	return &SimpleParser{}
}

func (p *SimpleParser) Parse(data []byte, fileName string) (*ParsedDocument, error) {
	if len(data) == 0 {
		return nil, errors.New("empty document data")
	}

	if strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		return p.parsePDF(data, fileName)
	}

	if strings.HasSuffix(strings.ToLower(fileName), ".txt") {
		return p.parseText(data, fileName)
	}

	return p.parseText(data, fileName)
}

func (p *SimpleParser) parsePDF(data []byte, fileName string) (*ParsedDocument, error) {
	text := p.extractTextFromPDF(data)
	if text == "" {
		return nil, errors.New("no text extracted from PDF")
	}

	chunks := p.chunkText(text, 1)
	return &ParsedDocument{
		FileName: fileName,
		Chunks:   chunks,
	}, nil
}

func (p *SimpleParser) parseText(data []byte, fileName string) (*ParsedDocument, error) {
	text := string(data)
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("empty text content")
	}

	chunks := p.chunkText(text, 1)
	return &ParsedDocument{
		FileName: fileName,
		Chunks:   chunks,
	}, nil
}

func (p *SimpleParser) extractTextFromPDF(data []byte) string {
	buf := bytes.NewBuffer(data)
	content := buf.String()

	lines := strings.Split(content, "\n")
	var result strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && isPrintable(line) {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (p *SimpleParser) chunkText(text string, pageNum int) []Chunk {
	const chunkSize = 1000
	const overlap = 100

	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return nil
	}

	var chunks []Chunk
	for i := 0; i < len(text); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunk := strings.TrimSpace(text[i:end])
		if chunk != "" {
			chunks = append(chunks, Chunk{
				Content: chunk,
				Page:    pageNum,
			})
		}

		if end >= len(text) {
			break
		}
	}

	return chunks
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
