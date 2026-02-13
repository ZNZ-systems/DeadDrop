package domain

import (
	"bufio"
	"os"
	"strings"
)

// FileResolver reads DNS TXT overrides from a file. Each line is
// "domain=value". The file is re-read on every LookupTXT call so
// records can be added at runtime (e.g., by e2e test scripts).
type FileResolver struct {
	path string
}

// NewFileResolver creates a FileResolver that reads from the given file path.
func NewFileResolver(path string) (*FileResolver, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &FileResolver{path: path}, nil
}

// LookupTXT returns all TXT record values for the given host.
func (r *FileResolver) LookupTXT(host string) ([]string, error) {
	f, err := os.Open(r.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && parts[0] == host {
			records = append(records, parts[1])
		}
	}
	return records, scanner.Err()
}
