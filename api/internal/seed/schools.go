package seed

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

type School struct {
	Name    string
	Jenjang string
	City    string
}

// LoadSchools reads a CSV with header name,jenjang,city; jenjang and city may be empty.
func LoadSchools(path string) ([]School, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is the operator's own CLI flag
	if err != nil {
		return nil, fmt.Errorf("read schools file: %w", err)
	}
	rows, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse schools file: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return parseSchoolRows(rows[1:])
}

func parseSchoolRows(rows [][]string) ([]School, error) {
	schools := make([]School, 0, len(rows))
	for i, row := range rows {
		if len(row) != 3 {
			return nil, fmt.Errorf("schools row %d: expected 3 columns, got %d", i+2, len(row))
		}
		name := strings.TrimSpace(row[0])
		if name == "" {
			return nil, fmt.Errorf("schools row %d: name is empty", i+2)
		}
		schools = append(schools, School{Name: name, Jenjang: strings.TrimSpace(row[1]), City: strings.TrimSpace(row[2])})
	}
	return schools, nil
}
