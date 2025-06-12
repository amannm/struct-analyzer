package core

import (
	"bytes"
	"encoding/json"
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
		expected, err := ReadJSON("testdata/expected.json")
		if err != nil {
			t.Fatal(err)
		}
		actual, err := ReadJSON(dest)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Errorf("expected: %s, actual: %s", expected, actual)
		}
	})
}

func ReadJSON(source string) ([]byte, error) {
	content, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	return json.Marshal(content)
}
