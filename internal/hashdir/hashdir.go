// Package hashdir computes deterministic content hashes of directory trees.
package hashdir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Hash returns "sha256:<hex>" over the dir tree. WalkDir visits files in
// lexical order, so the result is deterministic. Each file contributes its
// slash-separated relative path, file permissions, and content, NUL-separated,
// so renames, moves, chmod drift, and content edits change the hash.
func Hash(dir string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		io.WriteString(h, filepath.ToSlash(rel))
		h.Write([]byte{0})
		info, err := d.Info()
		if err != nil {
			return err
		}
		io.WriteString(h, fmt.Sprintf("%04o", info.Mode().Perm()))
		h.Write([]byte{0})
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return err
		}
		f.Close()
		h.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
