package analyzer

import (
	"testing"
)

func TestBasic(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		err := AnalyzeSourceRoot([]string{
			//"/Users/amannmalik/GolandProjects/opentelemetry-collector",
			//"/Users/amannmalik/GolandProjects/opentelemetry-collector-contrib",
			//"/Users/amannmalik/GolandProjects/blackfriday",
			//"/Users/amannmalik/GolandProjects/runc",
			//"/Users/amannmalik/GolandProjects/cgroups",
			//"/Users/amannmalik/GolandProjects/sys",
			//"/Users/amannmalik/GolandProjects/go-systemd",
			"/Users/amannmalik/GolandProjects/runtime-spec/specs-go",
		}, "analysis.json")
		if err != nil {
			t.Fatal(err)
		}
	})
}
