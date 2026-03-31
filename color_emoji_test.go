package fpdf_test

import (
	"bytes"
	"os"
	"testing"

	"codeberg.org/go-pdf/fpdf"
	"codeberg.org/go-pdf/fpdf/internal/example"
)

// TestColorBitmapEmoji tests that color bitmap emoji (CBDT/CBLC fonts)
// render as inline PNG images without panicking.
func TestColorBitmapEmoji(t *testing.T) {
	fontPath := example.FontFile("NotoColorEmoji.ttf")
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		t.Skip("NotoColorEmoji.ttf not available in font directory")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.AddUTF8Font("emoji", "", fontPath)
	pdf.SetFont("emoji", "", 14)

	// These emoji have CBDT bitmap glyphs
	pdf.Cell(100, 10, "\U0001F389\U0001F3C6\u2B50")

	if pdf.Err() {
		t.Fatalf("unexpected error after Cell: %v", pdf.Error())
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		t.Fatalf("unexpected error generating PDF: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("generated PDF is empty")
	}

	// Verify the PDF contains PNG image data (the emoji bitmaps)
	pdfContent := buf.String()
	if !bytes.Contains([]byte(pdfContent), []byte("/Subtype /Image")) {
		t.Error("PDF does not contain image objects - bitmap emoji may not have been rendered")
	}
}

// TestColorBitmapEmojiMixedFonts tests mixing a regular text font with a color emoji font.
func TestColorBitmapEmojiMixedFonts(t *testing.T) {
	emojiFontPath := example.FontFile("NotoColorEmoji.ttf")
	if _, err := os.Stat(emojiFontPath); os.IsNotExist(err) {
		t.Skip("NotoColorEmoji.ttf not available in font directory")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.AddUTF8Font("dejavu", "", example.FontFile("DejaVuSansCondensed.ttf"))
	pdf.AddUTF8Font("emoji", "", emojiFontPath)

	pdf.SetFont("dejavu", "", 14)
	pdf.Cell(0, 8, "Hello World! ")

	pdf.SetFont("emoji", "", 14)
	pdf.Cell(0, 8, "\U0001F389\U0001F3C6")

	if pdf.Err() {
		t.Fatalf("unexpected error: %v", pdf.Error())
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		t.Fatalf("unexpected error generating PDF: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("generated PDF is empty")
	}
}
