package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads .env into the process environment. Already-set OS variables are not overwritten.
// Order: first ".env" in the current working directory, then ".env" next to the executable (only keys still unset are applied from the second file).
func LoadDotEnv() {
	paths := []string{".env"}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), ".env"))
	}
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		_ = godotenv.Load(p)
	}
}
