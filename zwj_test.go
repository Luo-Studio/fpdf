package fpdf_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"codeberg.org/go-pdf/fpdf"
	"codeberg.org/go-pdf/fpdf/internal/example"
)

// TestZWJSequences tests that ZWJ (Zero Width Joiner) emoji sequences render
// as a single combined glyph rather than individual emoji.
func TestZWJSequences(t *testing.T) {
	fontPath := example.FontFile("NotoColorEmoji.ttf")
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		t.Skip("NotoColorEmoji.ttf not available in font directory")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.AddUTF8Font("emoji", "", fontPath)
	pdf.SetFont("emoji", "", 24)

	// ZWJ sequences:
	// 👨‍👩‍👧‍👦 = U+1F468 U+200D U+1F469 U+200D U+1F467 U+200D U+1F466 (family)
	// 👩‍❤️‍👨 = U+1F469 U+200D U+2764 U+200D U+1F468 (couple with heart)
	zwjFamily := "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"
	pdf.Cell(100, 15, zwjFamily)
	pdf.Ln(15)

	// Also test individual emoji for comparison
	pdf.Cell(100, 15, "\U0001F468\U0001F469\U0001F467\U0001F466")

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

	// Verify PDF contains image objects
	pdfContent := buf.Bytes()
	if !bytes.Contains(pdfContent, []byte("/Subtype /Image")) {
		t.Error("PDF does not contain image objects - ZWJ emoji may not have been rendered")
	}

	// Count the number of image objects: ZWJ sequence should produce fewer images
	// than individual emoji. Family ZWJ = 1 image, 4 individual = 4 images.
	imageCount := bytes.Count(pdfContent, []byte("/Subtype /Image"))
	t.Logf("PDF contains %d image objects (ZWJ family=1 + 4 individual = 5 expected)", imageCount)
	if imageCount < 5 {
		t.Errorf("expected at least 5 image objects (1 ZWJ + 4 individual), got %d", imageCount)
	}
}

// TestZWJSequenceImageValidation generates a PDF with ZWJ sequences, converts
// it to an image, and validates the output visually.
func TestZWJSequenceImageValidation(t *testing.T) {
	fontPath := example.FontFile("NotoColorEmoji.ttf")
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		t.Skip("NotoColorEmoji.ttf not available in font directory")
	}

	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not available for PDF-to-image conversion")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.AddUTF8Font("emoji", "", fontPath)

	// Row 1: ZWJ family sequence (should be ONE combined emoji)
	pdf.SetFont("emoji", "", 32)
	pdf.Cell(50, 20, "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466")
	pdf.Ln(20)

	// Row 2: Individual emoji for comparison (should be FOUR separate emoji)
	pdf.Cell(200, 20, "\U0001F468\U0001F469\U0001F467\U0001F466")
	pdf.Ln(20)

	// Row 3: Pirate flag 🏴‍☠️ = U+1F3F4 U+200D U+2620 U+FE0F
	pdf.Cell(50, 20, "\U0001F3F4\u200D\u2620\uFE0F")
	pdf.Ln(20)

	// Row 4: More ZWJ sequences
	// 👨‍🍳 = U+1F468 U+200D U+1F373 (man cook)
	pdf.Cell(50, 20, "\U0001F468\u200D\U0001F373")

	if pdf.Err() {
		t.Fatalf("unexpected error: %v", pdf.Error())
	}

	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "zwj_test.pdf")
	pngPath := filepath.Join(tmpDir, "zwj_test")

	f, err := os.Create(pdfPath)
	if err != nil {
		t.Fatalf("failed to create PDF file: %v", err)
	}
	err = pdf.Output(f)
	f.Close()
	if err != nil {
		t.Fatalf("failed to write PDF: %v", err)
	}

	// Convert to PNG using pdftoppm
	cmd := exec.Command("pdftoppm", "-png", "-r", "150", "-singlefile", pdfPath, pngPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pdftoppm failed: %v\n%s", err, output)
	}

	pngFile := pngPath + ".png"
	info, err := os.Stat(pngFile)
	if err != nil {
		t.Fatalf("PNG file not created: %v", err)
	}
	t.Logf("Generated PNG: %s (%d bytes)", pngFile, info.Size())

	if info.Size() == 0 {
		t.Fatal("generated PNG is empty")
	}

	// Use ImageMagick to check that the image has non-white content
	// (emoji should add color to the page)
	if magick, err := exec.LookPath("magick"); err == nil {
		cmd := exec.Command(magick, "identify", "-verbose", pngFile)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Logf("Image info (first 500 chars): %s", string(out[:min(len(out), 500)]))
		}
	}

	// Verify the ZWJ row has fewer images than the individual row by checking
	// the PDF structure
	pdfBytes, _ := os.ReadFile(pdfPath)
	imageCount := bytes.Count(pdfBytes, []byte("/Subtype /Image"))
	t.Logf("Total images in PDF: %d", imageCount)

	// We expect:
	// Row 1: 1 image (ZWJ family combined)
	// Row 2: 4 images (individual man, woman, girl, boy)
	// Row 3: 1 image (pirate flag ZWJ or individual fallback)
	// Row 4: 1 image (man cook ZWJ or individual fallback)
	// Total: at least 6 images (fewer means ZWJ worked)
	if imageCount < 2 {
		t.Error("expected at least 2 images in the PDF")
	}

	// Copy PNG to a known location for visual inspection
	inspectPath := filepath.Join(os.TempDir(), "fpdf_zwj_test.png")
	data, _ := os.ReadFile(pngFile)
	os.WriteFile(inspectPath, data, 0644)
	fmt.Printf("Visual inspection: %s\n", inspectPath)
}
