package exporter

import (
	"archive/zip"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// Berkas ODS adalah arsip zip berisi beberapa XML. Ditulis manual karena excelize
// tidak mendukung OpenDocument, dan formatnya cukup sederhana untuk tidak menambah dependensi.
const (
	odsMimeType = "application/vnd.oasis.opendocument.spreadsheet"

	odsManifest = `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">
 <manifest:file-entry manifest:full-path="/" manifest:version="1.2" manifest:media-type="` + odsMimeType + `"/>
 <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
</manifest:manifest>`

	odsContentHead = `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content` +
		` xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"` +
		` xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0"` +
		` xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"` +
		` office:version="1.2"><office:body><office:spreadsheet>`

	odsContentTail = `</office:spreadsheet></office:body></office:document-content>`
)

// writeODS menulis hasil query sebagai spreadsheet OpenDocument.
// Baris ditulis mengalir langsung ke dalam zip, jadi dataset besar tidak ditahan di memori.
func writeODS(outputPath, sheetName string, headers []string, rows *sql.Rows) (int64, error) {
	if sheetName == "" {
		sheetName = "Data"
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("gagal membuat file ODS: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// Spesifikasi ODF: entri "mimetype" harus pertama dan tanpa kompresi
	mimeEntry, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	if err != nil {
		return 0, err
	}
	if _, err := io.WriteString(mimeEntry, odsMimeType); err != nil {
		return 0, err
	}

	manifestEntry, err := zw.Create("META-INF/manifest.xml")
	if err != nil {
		return 0, err
	}
	if _, err := io.WriteString(manifestEntry, odsManifest); err != nil {
		return 0, err
	}

	content, err := zw.Create("content.xml")
	if err != nil {
		return 0, err
	}

	var sb strings.Builder
	sb.WriteString(odsContentHead)
	sb.WriteString(`<table:table table:name="`)
	sb.WriteString(odsEscape(sheetName))
	sb.WriteString(`">`)
	sb.WriteString(fmt.Sprintf(`<table:table-column table:number-columns-repeated="%d"/>`, len(headers)))
	writeODSRow(&sb, headers)
	if _, err := io.WriteString(content, sb.String()); err != nil {
		return 0, err
	}

	var count int64
	for rows.Next() {
		vals, err := scanRowStrings(rows, len(headers))
		if err != nil {
			continue
		}
		sb.Reset()
		writeODSRow(&sb, vals)
		if _, err := io.WriteString(content, sb.String()); err != nil {
			return 0, fmt.Errorf("gagal menulis baris ODS: %w", err)
		}
		count++
	}

	if _, err := io.WriteString(content, `</table:table>`+odsContentTail); err != nil {
		return 0, err
	}
	if err := zw.Close(); err != nil {
		return 0, fmt.Errorf("gagal menutup arsip ODS: %w", err)
	}
	return count, nil
}

func writeODSRow(sb *strings.Builder, vals []string) {
	sb.WriteString("<table:table-row>")
	for _, v := range vals {
		if v == "" {
			sb.WriteString("<table:table-cell/>")
			continue
		}
		sb.WriteString(`<table:table-cell office:value-type="string"><text:p>`)
		sb.WriteString(odsEscape(v))
		sb.WriteString("</text:p></table:table-cell>")
	}
	sb.WriteString("</table:table-row>")
}

func odsEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
