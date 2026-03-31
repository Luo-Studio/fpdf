// Copyright ©2023 The go-pdf Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fpdf

import "encoding/binary"

// ligature represents a GSUB ligature substitution: a sequence of glyph IDs
// that is replaced by a single result glyph ID.
type ligature struct {
	// components contains the remaining glyph IDs after the first one
	// (the first glyph is the key in the ligatureSet map).
	components []int
	resultGID  int
}

// parseGSUBTable parses the GSUB (Glyph Substitution) table to extract
// ligature substitutions. This is used to support ZWJ emoji sequences where
// multiple codepoints (joined by U+200D) map to a single combined glyph.
// Returns a map from first glyph ID to a list of ligatures.
func (utf *utf8FontFile) parseGSUBTable() map[int][]ligature {
	desc, ok := utf.tableDescriptions["GSUB"]
	if !ok {
		return nil
	}

	data := utf.fileReader.array
	gsub := desc.position

	if gsub+10 > len(data) {
		return nil
	}

	// GSUB header: version(4) + scriptListOffset(2) + featureListOffset(2) + lookupListOffset(2)
	lookupListOff := gsub + int(binary.BigEndian.Uint16(data[gsub+8:gsub+10]))
	if lookupListOff+2 > len(data) {
		return nil
	}

	numLookups := int(binary.BigEndian.Uint16(data[lookupListOff : lookupListOff+2]))
	result := make(map[int][]ligature)

	for i := 0; i < numLookups; i++ {
		offIdx := lookupListOff + 2 + i*2
		if offIdx+2 > len(data) {
			break
		}
		lookupOff := lookupListOff + int(binary.BigEndian.Uint16(data[offIdx:offIdx+2]))
		if lookupOff+6 > len(data) {
			continue
		}

		lookupType := int(binary.BigEndian.Uint16(data[lookupOff : lookupOff+2]))
		// We only care about type 4 = Ligature Substitution
		if lookupType != 4 {
			continue
		}

		numSubtables := int(binary.BigEndian.Uint16(data[lookupOff+4 : lookupOff+6]))
		for si := 0; si < numSubtables; si++ {
			stIdx := lookupOff + 6 + si*2
			if stIdx+2 > len(data) {
				break
			}
			subtableOff := lookupOff + int(binary.BigEndian.Uint16(data[stIdx:stIdx+2]))
			utf.parseLigatureSubtable(data, subtableOff, result)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// parseLigatureSubtable parses a single LigatureSubst subtable (format 1).
func (utf *utf8FontFile) parseLigatureSubtable(data []byte, off int, result map[int][]ligature) {
	if off+6 > len(data) {
		return
	}

	format := int(binary.BigEndian.Uint16(data[off : off+2]))
	if format != 1 {
		return
	}

	covOff := off + int(binary.BigEndian.Uint16(data[off+2:off+4]))
	ligSetCount := int(binary.BigEndian.Uint16(data[off+4 : off+6]))

	// Parse coverage table to get first glyphs
	coverageGlyphs := utf.parseCoverageTable(data, covOff)
	if len(coverageGlyphs) != ligSetCount {
		return
	}

	for i := 0; i < ligSetCount; i++ {
		lsIdx := off + 6 + i*2
		if lsIdx+2 > len(data) {
			break
		}
		lsOff := off + int(binary.BigEndian.Uint16(data[lsIdx:lsIdx+2]))
		if lsOff+2 > len(data) {
			continue
		}

		firstGlyph := coverageGlyphs[i]
		ligCount := int(binary.BigEndian.Uint16(data[lsOff : lsOff+2]))

		for li := 0; li < ligCount; li++ {
			liIdx := lsOff + 2 + li*2
			if liIdx+2 > len(data) {
				break
			}
			ligOff := lsOff + int(binary.BigEndian.Uint16(data[liIdx:liIdx+2]))
			if ligOff+4 > len(data) {
				continue
			}

			ligGlyph := int(binary.BigEndian.Uint16(data[ligOff : ligOff+2]))
			compCount := int(binary.BigEndian.Uint16(data[ligOff+2 : ligOff+4]))
			if compCount < 2 {
				continue
			}

			components := make([]int, compCount-1)
			valid := true
			for ci := 0; ci < compCount-1; ci++ {
				cIdx := ligOff + 4 + ci*2
				if cIdx+2 > len(data) {
					valid = false
					break
				}
				components[ci] = int(binary.BigEndian.Uint16(data[cIdx : cIdx+2]))
			}
			if !valid {
				continue
			}

			result[firstGlyph] = append(result[firstGlyph], ligature{
				components: components,
				resultGID:  ligGlyph,
			})
		}
	}
}

// parseCoverageTable parses a Coverage table and returns the list of glyph IDs.
func (utf *utf8FontFile) parseCoverageTable(data []byte, off int) []int {
	if off+4 > len(data) {
		return nil
	}

	format := int(binary.BigEndian.Uint16(data[off : off+2]))

	switch format {
	case 1: // Coverage Format 1: list of glyph IDs
		count := int(binary.BigEndian.Uint16(data[off+2 : off+4]))
		glyphs := make([]int, 0, count)
		for i := 0; i < count; i++ {
			idx := off + 4 + i*2
			if idx+2 > len(data) {
				break
			}
			glyphs = append(glyphs, int(binary.BigEndian.Uint16(data[idx:idx+2])))
		}
		return glyphs

	case 2: // Coverage Format 2: ranges
		rangeCount := int(binary.BigEndian.Uint16(data[off+2 : off+4]))
		var glyphs []int
		for i := 0; i < rangeCount; i++ {
			idx := off + 4 + i*6
			if idx+6 > len(data) {
				break
			}
			startGlyph := int(binary.BigEndian.Uint16(data[idx : idx+2]))
			endGlyph := int(binary.BigEndian.Uint16(data[idx+2 : idx+4]))
			for g := startGlyph; g <= endGlyph; g++ {
				glyphs = append(glyphs, g)
			}
		}
		return glyphs
	}

	return nil
}
