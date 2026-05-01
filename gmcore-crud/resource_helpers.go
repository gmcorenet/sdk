package gmcorecrud

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	gmcoreform "gmcore-form"
)

var csvFormulaPattern = regexp.MustCompile(`^[=+\-@\t\r]+`)

func csvEscapeField(value string) string {
	value = strings.TrimSpace(value)
	if csvFormulaPattern.MatchString(value) {
		return "'" + value
	}
	return value
}

type LayoutFieldMeta struct {
	Name       string   `json:"name"`
	Hidden     bool     `json:"hidden"`
	ColSpan    int      `json:"col_span"`
	Widget     string   `json:"widget"`
	Height     int      `json:"height"`
	Validation []string `json:"validation,omitempty"`
}

type LayoutMeta struct {
	Fields []LayoutFieldMeta `json:"fields"`
}

func EffectiveFormDefinition(resourceName, titleKey string, form gmcoreform.Definition, cfg Config) gmcoreform.Definition {
	if len(form.Fields) > 0 {
		return form
	}
	fields := make([]gmcoreform.Field, 0, len(cfg.Fields))
	for _, field := range cfg.Fields {
		formField := gmcoreform.Field{
			Name:        field.Name,
			Label:       field.Label,
			LabelKey:    field.LabelKey,
			Type:        field.Type,
			Widget:      defaultWidgetForField(field),
			Required:    field.Required,
			Placeholder: field.Placeholder,
			HelpKey:     field.HelpKey,
			Relation:    field.Relation,
			ReadOnly:    !field.Writable,
			AutoManaged: isAutoManagedField(field.Name),
		}
		if relation, ok := RelationByName(cfg.Relations, field.Relation); ok {
			formField.AsyncOptions = relation.Async
			formField.AsyncLimit = relation.AsyncLimit
			formField.AsyncDebounce = relation.AsyncDebounce
			formField.LoadAllLimit = relation.LoadAllLimit
			formField.ValueField = relation.ValueField
			formField.DisplayField = relation.DisplayField
			if relation.Widget != "" {
				formField.Widget = relation.Widget
			}
			if relation.Type == RelationHasMany || relation.Type == RelationManyToMany {
				formField.Multiple = true
				if relation.Widget == "" {
					formField.Widget = "multiselect"
				}
			}
		}
		if formField.AutoManaged {
			formField.ReadOnly = true
		}
		if field.Name == "password_hash" {
			formField.WriteOnly = true
		}
		fields = append(fields, formField)
	}
	return gmcoreform.Definition{
		Name:    resourceName,
		Title:   titleKey,
		Fields:  fields,
		Buttons: nil,
	}
}

func ApplyLayoutMeta(form gmcoreform.Definition, meta LayoutMeta) gmcoreform.Definition {
	if len(meta.Fields) == 0 {
		return form
	}
	fieldMap := map[string]gmcoreform.Field{}
	for _, field := range form.Fields {
		fieldMap[field.Name] = field
	}
	out := make([]gmcoreform.Field, 0, len(form.Fields))
	seen := map[string]struct{}{}
	for _, current := range meta.Fields {
		field, ok := fieldMap[current.Name]
		if !ok {
			continue
		}
		if current.ColSpan > 0 {
			field.ColSpan = current.ColSpan
		}
		if current.Widget != "" {
			field.Widget = current.Widget
		}
		if current.Height > 0 {
			field.Height = current.Height
		}
		if current.Validation != nil {
			field.Validation = append([]string(nil), current.Validation...)
		}
		field.Hidden = current.Hidden
		out = append(out, field)
		seen[current.Name] = struct{}{}
	}
	for _, field := range form.Fields {
		if _, ok := seen[field.Name]; ok {
			continue
		}
		out = append(out, field)
	}
	form.Fields = out
	return form
}

func ExportRecords(w http.ResponseWriter, resourceName string, cfg Config, action string, records []Record) error {
	columns := cfg.Index.Columns
	if len(columns) == 0 {
		for _, field := range cfg.Fields {
			if field.Visible {
				columns = append(columns, IndexColumn{
					Field: field.Name,
					Label: field.Label,
				})
			}
		}
	}
	switch action {
	case "export_text":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.txt"`, resourceName))
		for _, record := range records {
			for _, col := range columns {
				_, _ = fmt.Fprintf(w, "%s: %s\n", col.Label, formatCellValue(col, record))
			}
			_, _ = fmt.Fprintln(w, "---")
		}
		return nil
	case "export_csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.csv"`, resourceName))
		writer := csv.NewWriter(w)
		header := make([]string, 0, len(columns))
		for _, col := range columns {
			header = append(header, csvEscapeField(col.Label))
		}
		if err := writer.Write(header); err != nil {
			return err
		}
		for _, record := range records {
			row := make([]string, 0, len(columns))
			for _, col := range columns {
				row = append(row, csvEscapeField(formatCellValue(col, record)))
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case "export_pdf":
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.pdf"`, resourceName))
		_, err := w.Write(buildSimpleTablePDF(columns, records))
		return err
	default:
		return fmt.Errorf("unknown bulk action %q", action)
	}
}

func defaultWidgetForField(field Field) string {
	switch field.Type {
	case "email":
		return "email"
	case "password", "password_hash":
		return "password"
	case "datetime":
		return "datetime-local"
	case "date":
		return "date"
	case "int", "integer", "float", "number":
		return "number"
	case "json", "array":
		return "textarea"
	default:
		return "text"
	}
}

func isAutoManagedField(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "created_at", "updated_at", "deleted_at", "createdat", "updatedat", "deletedat":
		return true
	default:
		return false
	}
}

func buildSimpleTablePDF(columns []IndexColumn, records []Record) []byte {
	lines := make([]string, 0, len(records)+2)
	header := make([]string, 0, len(columns))
	for _, col := range columns {
		header = append(header, col.Label)
	}
	lines = append(lines, strings.Join(header, " | "))
	lines = append(lines, strings.Repeat("-", 100))
	for _, record := range records {
		row := make([]string, 0, len(columns))
		for _, col := range columns {
			row = append(row, formatCellValue(col, record))
		}
		lines = append(lines, strings.Join(row, " | "))
	}
	text := strings.Join(lines, "\n")
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	stream := "BT /F1 10 Tf 36 806 Td 12 TL " + strings.ReplaceAll(text, "\n", " Tj T* (")
	stream = "(" + stream + ") Tj ET"
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := []int{}
	writeObj := func(id int, body string) {
		offsets = append(offsets, buf.Len())
		_, _ = fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", id, body)
	}
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Count 1 /Kids [3 0 R] >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>")
	writeObj(4, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	writeObj(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	xref := buf.Len()
	_, _ = fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, off := range offsets {
		_, _ = fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	_, _ = fmt.Fprintf(&buf, "trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(offsets)+1, xref)
	return buf.Bytes()
}
