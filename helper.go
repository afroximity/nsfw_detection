package nsfw

import (
	"os"
	"path/filepath"
	"strings"
)

func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func envStr(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func looksLikeSavedModel(dir string) bool {
	// tfgo loads a TF SavedModel; the canonical marker is saved_model.pb
	return fileExists(filepath.Join(dir, "saved_model.pb"))
}
