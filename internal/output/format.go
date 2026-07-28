// Package output provides result formatting and error writing utilities.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Format controls output serialization.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatPlain Format = "plain"
)

// ParseFormat parses a format string, defaulting to JSON on unrecognized input.
func ParseFormat(s string) Format {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return FormatJSON
	case "table":
		return FormatTable
	case "plain":
		return FormatPlain
	default:
		return FormatJSON
	}
}

// WriteResult writes data to w in the requested format.
func WriteResult(w io.Writer, data any, format Format) error {
	switch format {
	case FormatJSON:
		return WriteJSON(w, data)
	case FormatTable:
		return WriteTable(w, data)
	case FormatPlain:
		return WritePlain(w, data)
	default:
		return WriteJSON(w, data)
	}
}

// WriteJSON writes data as pretty-printed JSON.
func WriteJSON(w io.Writer, data any) error {
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	buf = append(buf, '\n')
	_, err = w.Write(buf)
	return err
}

// WriteTable writes data as a formatted table using tabwriter.
// data should be a [][]string where the first row is the header.
func WriteTable(w io.Writer, data any) error {
	rows, ok := data.([][]string)
	if !ok {
		return WriteJSON(w, data)
	}
	if len(rows) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	for i, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
		if i == 0 {
			// header separator
			sep := make([]string, len(row))
			for j := range row {
				sep[j] = strings.Repeat("-", len(row[j]))
			}
			if _, err := fmt.Fprintln(tw, strings.Join(sep, "\t")); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}

// WritePlain writes data using the default string representation.
func WritePlain(w io.Writer, data any) error {
	_, err := fmt.Fprintln(w, data)
	return err
}

// WriteError writes an error as JSON to w with the format {"error":"...","code":"..."}.
func WriteError(w io.Writer, err error, code string) {
	enc := json.NewEncoder(w)
	_ = enc.Encode(struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}{
		Error: err.Error(),
		Code:  code,
	})
}
