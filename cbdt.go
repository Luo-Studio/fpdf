// Copyright ©2023 The go-pdf Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fpdf

import "encoding/binary"

// bitmapGlyph represents a color bitmap glyph extracted from CBDT/CBLC tables.
type bitmapGlyph struct {
	Width    int    // bitmap width in pixels
	Height   int    // bitmap height in pixels
	BearingX int    // horizontal bearing
	BearingY int    // vertical bearing
	Advance  int    // horizontal advance
	PNGData  []byte // raw PNG image data
}

// parseCBLCTable parses the Color Bitmap Location Table (CBLC) and Color Bitmap
// Data Table (CBDT) to extract PNG bitmaps for color emoji glyphs.
// Returns a map of glyph ID -> bitmapGlyph.
func (utf *utf8FontFile) parseCBLCTable() map[int]*bitmapGlyph {
	cblcDesc, hasCBLC := utf.tableDescriptions["CBLC"]
	cbdtDesc, hasCBDT := utf.tableDescriptions["CBDT"]
	if !hasCBLC || !hasCBDT {
		return nil
	}

	data := utf.fileReader.array
	cblc := cblcDesc.position
	cbdt := cbdtDesc.position

	if cblc+8 > len(data) {
		return nil
	}

	numSizes := int(binary.BigEndian.Uint32(data[cblc+4 : cblc+8]))
	if numSizes == 0 {
		return nil
	}

	result := make(map[int]*bitmapGlyph)

	// Pick the largest strike (highest ppem) for best quality
	bestStrike := -1
	bestPPEM := 0
	for s := 0; s < numSizes; s++ {
		rec := cblc + 8 + s*48
		if rec+48 > len(data) {
			break
		}
		bitDepth := int(data[rec+46])
		ppemY := int(data[rec+45])
		// We want 32-bit depth (RGBA) bitmaps
		if bitDepth == 32 && ppemY > bestPPEM {
			bestPPEM = ppemY
			bestStrike = s
		}
	}

	if bestStrike < 0 {
		return nil
	}

	rec := cblc + 8 + bestStrike*48
	indexSubTableArrayOffset := int(binary.BigEndian.Uint32(data[rec : rec+4]))
	numberOfIndexSubTables := int(binary.BigEndian.Uint32(data[rec+8 : rec+12]))

	for ist := 0; ist < numberOfIndexSubTables; ist++ {
		istOff := cblc + indexSubTableArrayOffset + ist*8
		if istOff+8 > len(data) {
			break
		}
		firstGlyph := int(binary.BigEndian.Uint16(data[istOff : istOff+2]))
		lastGlyph := int(binary.BigEndian.Uint16(data[istOff+2 : istOff+4]))
		addOffset := int(binary.BigEndian.Uint32(data[istOff+4 : istOff+8]))

		subtableOff := cblc + indexSubTableArrayOffset + addOffset
		if subtableOff+8 > len(data) {
			continue
		}
		indexFormat := int(binary.BigEndian.Uint16(data[subtableOff : subtableOff+2]))
		imageFormat := int(binary.BigEndian.Uint16(data[subtableOff+2 : subtableOff+4]))
		imageDataOffset := int(binary.BigEndian.Uint32(data[subtableOff+4 : subtableOff+8]))

		// We support index format 1 (variable-size) with image format 17 (small metrics + PNG)
		if indexFormat == 1 && imageFormat == 17 {
			numGlyphs := lastGlyph - firstGlyph + 1
			for g := 0; g < numGlyphs; g++ {
				offIdx := subtableOff + 8 + g*4
				if offIdx+8 > len(data) {
					break
				}
				off1 := int(binary.BigEndian.Uint32(data[offIdx : offIdx+4]))
				glyphDataOff := cbdt + imageDataOffset + off1
				if glyphDataOff+9 > len(data) {
					continue
				}

				// Format 17: smallGlyphMetrics (5 bytes) + uint32 dataLen + PNG data
				height := int(data[glyphDataOff])
				width := int(data[glyphDataOff+1])
				bearingX := int(int8(data[glyphDataOff+2]))
				bearingY := int(int8(data[glyphDataOff+3]))
				advance := int(data[glyphDataOff+4])
				dataLen := int(binary.BigEndian.Uint32(data[glyphDataOff+5 : glyphDataOff+9]))

				pngStart := glyphDataOff + 9
				pngEnd := pngStart + dataLen
				if pngEnd > len(data) || dataLen == 0 {
					continue
				}

				// Verify PNG magic bytes
				if data[pngStart] != 0x89 || data[pngStart+1] != 'P' || data[pngStart+2] != 'N' || data[pngStart+3] != 'G' {
					continue
				}

				pngData := make([]byte, dataLen)
				copy(pngData, data[pngStart:pngEnd])

				gid := firstGlyph + g
				result[gid] = &bitmapGlyph{
					Width:    width,
					Height:   height,
					BearingX: bearingX,
					BearingY: bearingY,
					Advance:  advance,
					PNGData:  pngData,
				}
			}
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
