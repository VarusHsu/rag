package parser

import (
	"testing"
)

func TestParseText(t *testing.T) {
	p := NewSimpleParser()
	data := []byte("Hello world. This is a test document.")

	doc, err := p.Parse(data, "test.txt")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if doc.FileName != "test.txt" {
		t.Fatalf("expected filename test.txt, got %s", doc.FileName)
	}

	if len(doc.Chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	if doc.Chunks[0].Content != "Hello world. This is a test document." {
		t.Fatalf("unexpected content: %s", doc.Chunks[0].Content)
	}
}

func TestParsingEmptyDocument(t *testing.T) {
	p := NewSimpleParser()
	_, err := p.Parse([]byte(""), "empty.txt")
	if err == nil {
		t.Fatal("expected error for empty document")
	}
}

func TestChunkText(t *testing.T) {
	p := NewSimpleParser()
	longText := ""
	for i := 0; i < 2000; i++ {
		longText += "word "
	}

	chunks := p.chunkText(longText, 1)
	if len(chunks) < 2 {
		t.Fatal("expected multiple chunks for long text")
	}
}
