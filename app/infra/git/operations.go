package git

import (
	"fmt"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/storage/memory"
	"io"
	"os"
	"strings"
	"superspreader/app/domain"
)

const GIT_DEPTH = 1

func CloneGit(repo domain.GitRepository, config domain.ProviderConfig) (*git.Repository, error) {
	auth := &http.BasicAuth{
		Username: config.Username,
		Password: config.Token,
	}
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:   repo.Url,
		Depth: GIT_DEPTH,
		Auth:  auth,
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}

func IsValidForSuperspreader(repo *git.Repository, config domain.Config) (bool, error) {

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

	content, err := io.ReadAll(reader)
	if err != nil {
		return false, err
	}
	contentStr := string(content)

	return strings.Contains(contentStr, config.Identifier.Content), nil
}

func CopyFiles(repo *git.Repository, config domain.FileConfig) error {

	for _, file := range config.Include {

		files, err := getAllFilenames(file)

		if err != nil {
			return err
		}

		fmt.Printf("files: %v\n", files)
	}

	return nil
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
