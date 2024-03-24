package log

import (
	"compress/gzip"
	"os"
	"strings"
)

// GzipLogFile convert and replace log file from text format to gzip
// compressed format.
func GzipLogFile(pathToFile string) error {
	// Get all the bytes from the file.
	content, err := os.ReadFile(pathToFile) // #nosec:G304, file inclusion via variable.
	if err != nil {
		return err
	}

	// Replace .log extension with .gz extension.
	newExt := strings.ReplaceAll(pathToFile, ".log", ".gz")

	// Open file for writing.
	gf, err := os.OpenFile(newExt, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644) // #nosec:G304,G302, file inclusion & perms
	if err != nil {
		return err
	}
	defer gf.Close() // #nosec:G307, error on close is not critical here

	// Write compressed data.
	w := gzip.NewWriter(gf)
	defer w.Close()
	if _, err = w.Write(content); err != nil {
		os.Remove(newExt) // #nosec:G104, not important error to handle

		return err
	}

	return os.Remove(pathToFile)
}
