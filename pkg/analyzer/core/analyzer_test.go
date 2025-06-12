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
		err := AnalyzeRepositories([]string{
			"https://github.com/opencontainers/runtime-spec",
		}, dest)
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
