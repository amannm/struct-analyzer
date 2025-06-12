package analyzer

import (
	"path/filepath"
	"testing"
)

func TestBasic(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "analysis.json")
		err := AnalyzeRemoteRepos([]string{
			"https://github.com/opencontainers/runtime-spec",
		}, dest)
		if err != nil {
			t.Fatal(err)
		}
	})
}
