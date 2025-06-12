package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBasic(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		dest := filepath.Join("testdata", "actual.json")
		_ = os.RemoveAll(dest)
		err := AnalyzeRepositories([]string{
			"https://github.com/open-telemetry/opentelemetry-collector",
			//"https://github.com/open-telemetry/opentelemetry-collector-contrib",
			//"https://github.com/russross/blackfriday",
			//"https://github.com/opencontainers/runc",
			//"https://github.com/containerd/cgroups",
			//"https://github.com/golang/sys",
			//"https://github.com/coreos/go-systemd",
			"https://github.com/opencontainers/runtime-spec",
		}, dest)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile("testdata/expected.json")
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Errorf("expected: %s, actual: %s", expected, actual)
		}
	})
}
