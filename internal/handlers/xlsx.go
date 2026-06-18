package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// Minimal single-sheet .xlsx writer using only the standard library (an xlsx is
// a ZIP of XML parts). Values are written as inline strings, so no shared-string
// table or styling is needed — enough for tabular report exports with Vietnamese
// (UTF-8) text. Avoids pulling a third-party Excel dependency.

func xlsxCol(i int) string {
	name := ""
	for i >= 0 {
		name = string(rune('A'+i%26)) + name
		i = i/26 - 1
	}
	return name
}

func xlsxEsc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func xlsxCell(col, row int, val string) string {
	return fmt.Sprintf(`<c r="%s%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, xlsxCol(col), row, xlsxEsc(val))
}

// writeXLSX streams a one-sheet workbook (header row + data rows) to out.
func writeXLSX(out io.Writer, sheetName string, headers []string, rows [][]string) error {
	zw := zip.NewWriter(out)
	add := func(name, content string) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(f, content)
		return err
	}

	var sd bytes.Buffer
	sd.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sd.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	emit := func(rowIdx int, cells []string) {
		sd.WriteString(fmt.Sprintf(`<row r="%d">`, rowIdx))
		for ci, v := range cells {
			sd.WriteString(xlsxCell(ci, rowIdx, v))
		}
		sd.WriteString(`</row>`)
	}
	emit(1, headers)
	for i, r := range rows {
		emit(i+2, r)
	}
	sd.WriteString(`</sheetData></worksheet>`)

	parts := []struct{ name, content string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
			`</Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
			fmt.Sprintf(`<sheets><sheet name="%s" sheetId="1" r:id="rId1"/></sheets>`, xlsxEsc(sheetName)) +
			`</workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`</Relationships>`},
		{"xl/worksheets/sheet1.xml", sd.String()},
	}
	for _, p := range parts {
		if err := add(p.name, p.content); err != nil {
			_ = zw.Close()
			return err
		}
	}
	return zw.Close()
}
