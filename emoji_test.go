package fpdf_test

import (
	"bytes"
	"testing"

	"codeberg.org/go-pdf/fpdf"
	"codeberg.org/go-pdf/fpdf/internal/example"
)

// TestSupplementaryPlaneChars tests that characters above U+FFFF (supplementary planes)
// do not cause a panic. DejaVuSansCondensed has cmap format 12 subtables and contains
// characters in the U+10300 range (Old Italic).
func TestSupplementaryPlaneChars(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.AddUTF8Font("dejavu", "", example.FontFile("DejaVuSansCondensed.ttf"))
	pdf.SetFont("dejavu", "", 14)

	// U+10300 is Old Italic Letter A - present in DejaVuSansCondensed's cmap format 12
	// This should not panic even though the codepoint > 0xFFFF
	pdf.Cell(100, 10, "Hello \U00010300\U00010301\U00010302 World")

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
}

// TestEmojiCharacters tests emoji characters (U+1F389, U+2B50, U+1F3C6).
// Even if the font doesn't have glyphs for these exact codepoints, the library
// must not panic.
func TestEmojiCharacters(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.AddUTF8Font("dejavu", "", example.FontFile("DejaVuSansCondensed.ttf"))
	pdf.SetFont("dejavu", "", 14)

	// These emoji codepoints are above U+FFFF. The font may not have glyphs
	// but the library must handle them gracefully without panicking.
	pdf.Cell(100, 10, "Emoji: 🎉⭐🏆")

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
}

// TestMixedBMPAndSupplementary tests a mix of BMP and supplementary plane characters.
func TestMixedBMPAndSupplementary(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.AddUTF8Font("dejavu", "", example.FontFile("DejaVuSansCondensed.ttf"))
	pdf.SetFont("dejavu", "", 14)

	// Mix of ASCII, BMP (Cyrillic), and supplementary plane chars
	pdf.Cell(100, 10, "Hello Мир 🎉 World \U00010300")

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
}
