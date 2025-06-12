package analyzer

import (
	"os"
	"strings"

	git "github.com/go-git/go-git/v5"
)

// cloneRepository clones the repository at the given URI into a temporary
// directory. It returns the path to the cloned repository and a cleanup
// function that should be called when the caller is done with the directory.
func cloneRepository(uri string) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "repo-*")
	if err != nil {
		return "", nil, err
	}
	_, err = git.PlainClone(tmp, false, &git.CloneOptions{URL: uri, Depth: 1})
	if err != nil {
		os.RemoveAll(tmp)
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }
	return tmp, cleanup, nil
}

// isRemote determines whether a path looks like a remote git URI.
func isRemote(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "git@")
}
