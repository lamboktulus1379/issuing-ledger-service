package configuration

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnvFromFile loads KEY=VALUE pairs from one or more files (e.g., config.env, .env)
// It ignores lines starting with # and empty lines. Existing env vars are not overridden.
func LoadEnvFromFile(paths ...string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Support KEY=VALUE and KEY="VALUE"
			if idx := strings.Index(line, "="); idx != -1 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				val = strings.Trim(val, "\"'")
				if key != "" {
					if _, exists := os.LookupEnv(key); !exists {
						_ = os.Setenv(key, val)
					}
				}
			}
		}
		_ = f.Close()
	}
}
