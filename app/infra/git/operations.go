package git

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/storage/memory"

	"go.iain.rocks/boneclone/app/domain"
)

const GIT_DEPTH = 1

func CloneGit(repo domain.GitRepository, config domain.ProviderConfig) (*git.Repository, billy.Filesystem, error) {
	fs := memfs.New()
	auth := &http.BasicAuth{
		Username: config.Username,
		Password: config.Token,
	}
	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:   repo.Url,
		Depth: GIT_DEPTH,
		Auth:  auth,
		Bare:  false,
	})

	if err != nil {
		return nil, nil, err
	}

	return r, fs, nil
}

func IsValidForBoneClone(repo *git.Repository, config domain.Config) (bool, error) {

	headRef, err := repo.Head()
	if err != nil {
		return false, err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())

	if err != nil {
		return false, err
	}
	tree, err := headCommit.Tree()
	if err != nil {
		return false, err
	}

	file, err := tree.File(config.Identifier.Filename)
	if err != nil {
		if err == object.ErrFileNotFound {
			return false, nil
		} else {
			return false, err
		}
	}

	// 6. Get the Blob object and read its content
	blob, err := repo.BlobObject(file.Hash)
	if err != nil {
		return false, err
	}

	reader, err := blob.Reader()
	if err != nil {
		return false, err
	}
	defer reader.Close()

	// Maybe change to io.Copy
	content, err := io.ReadAll(reader)
	if err != nil {
		return false, err
	}
	contentStr := string(content)

	return strings.Contains(contentStr, config.Identifier.Content), nil
}

func CopyFiles(
	repo *git.Repository,
	fs billy.Filesystem,
	config domain.FileConfig,
	provider domain.ProviderConfig,
) error {

	worktree, err := repo.Worktree()

	if err != nil {
		return err
	}

	for _, definedFile := range config.Include {

		files, err := getAllFilenames(definedFile)

		if err != nil {
			return err
		}

		for _, file := range files {

			if isExcluded(file, config.Exclude) {
				continue
			}

			parts := strings.Split(file, "/")
			parts = parts[:len(parts)-1]
			directory := strings.Join(parts, "/")
			_, err := fs.Lstat(directory)

			if err != nil {
				if err == os.ErrNotExist {
					err := fs.MkdirAll(directory, 0755)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}

			f, err := fs.Create(file)
			if err != nil {
				return err
			}
			defer f.Close()

			content, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			_, err = f.Write(content)
			_, err = worktree.Add(file)
			if err != nil {
				return err
			}
		}
		_, err = worktree.Commit("Updated via boneclone", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "boneclone",
				Email: "boneclone@example.com",
				When:  time.Now(),
			},
		})
		err = repo.Push(&git.PushOptions{
			Auth: &http.BasicAuth{Username: provider.Username, Password: provider.Token},
		})
		if err != nil {
			if err == git.NoErrAlreadyUpToDate {
				return nil
			}
			return err
		}
	}

	return nil
}

func isExcluded(filename string, excluded []string) bool {
	for _, excludedFile := range excluded {
		if filename == excludedFile {
			return true
		}
	}
	return false
}

func getAllFilenames(filename string) ([]string, error) {

	output := []string{}

	stat, err := os.Stat(filename)

	if err != nil {
		return []string{}, err
	}

	if stat.IsDir() {
		files, suberr := os.ReadDir(filename)

		if suberr != nil {
			return []string{}, suberr
		}

		for _, file := range files {
			fileLocation := fmt.Sprintf("%s/%s", filename, file.Name())
			if file.IsDir() {
				subFiles, suberr := getAllFilenames(fileLocation)

				if suberr != nil {
					return []string{}, suberr
				}

				output = append(output, subFiles...)
			} else {
				output = append(output, fileLocation)
			}
		}

	} else {
		output = append(output, filename)
	}

	return output, nil
}
