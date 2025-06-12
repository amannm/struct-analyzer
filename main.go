package struct_analyzer

import (
	"log"
	"os"
	"struct-analyzer/pkg/analyzer"
)

func main() {
	err := analyzer.AnalyzeRemoteRepos([]string{
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
		log.Fatal(err)
	}
	os.Exit(0)
}
