package analyzer

import (
	"testing"
)

func TestBasic(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := AnalyzeSourceRoot([]string{
			"/Users/amannmalik/GolandProjects/opentelemetry-collector",
			"/Users/amannmalik/GolandProjects/opentelemetry-collector-contrib",
		}, "analysis.json")
		if err != nil {
			t.Fatal(err)
		}
	})
}
