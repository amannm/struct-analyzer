package struct_analyzer

import (
	"log"
	"os"
	"struct-analyzer/pkg/analyzer/core"
)

func main() {
	var repoUris = os.Args[1:]
	err := core.AnalyzeRepositories(repoUris, "analysis.json")
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
