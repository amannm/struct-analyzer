package gitutil

import (
	"io"
	"os"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// WalkRemoteBlobs clones the repository at remoteURI into a temporary
// directory and invokes visitor for every blob found in the tree of the
// default branch. The temporary directory is removed after the walk
// completes.
func WalkRemoteBlobs(remoteURI string, visitor func(path string, contents []byte) error) error {
	tempDir, err := os.MkdirTemp("", "gitwalk-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{URL: remoteURI})
	if err != nil {
		return err
	}

	ref, err := repo.Head()
	if err != nil {
		return err
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	return tree.Files().ForEach(func(f *object.File) error {
		r, err := f.Reader()
		if err != nil {
			return err
		}
		defer r.Close()
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		return visitor(f.Name, data)
	})
}
