package struct_analyzer

import (
	"log"
	"os"
	"struct-analyzer/pkg/analyzer"
)

func main() {
	var repoUris = os.Args[1:]
	err := analyzer.AnalyzeRepositories(repoUris, "analysis.json")
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
