package git

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v6"
	gogitcfg "github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/storage/memory"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"

	"go.iain.rocks/boneclone/app/domain"
)

const (
	DefaultCommitterName  = "boneclone"
	DefaultCommitterEmail = "boneclone@example.org"
	GitDepth              = 1
)

// GitOperations defines the operations BoneClone needs for git interactions.
// This enables testability and future swapping/mocking of git behavior.
// NOTE: The canonical interface lives in the domain package. This duplicate
// definition remains only for historical context and should not be referenced.
type GitOperations interface {
	CloneGit(repo domain.GitRepository, config domain.ProviderConfig) (*git.Repository, billy.Filesystem, error)
	IsValidForBoneClone(repo *git.Repository, config domain.Config) (bool, error)
	CopyFiles(repo *git.Repository, fs billy.Filesystem, config domain.Config, provider domain.ProviderConfig, targetBranch string) error
}

// Operations is the default implementation of git operations using go-git and memfs.
type Operations struct{}

// NewOperations creates a new default Operations implementation, returned as a domain.GitOperations.
func NewOperations() domain.GitOperations { return &Operations{} }

// DefaultOps is the package-level default used by the wrapper functions to
// maintain backward compatibility with existing callers.
var DefaultOps domain.GitOperations = NewOperations()

// Method implementations
func (o *Operations) CloneGit(repo domain.GitRepository, config domain.ProviderConfig) (*git.Repository, billy.Filesystem, error) {
	fs := memfs.New()
	auth := &http.BasicAuth{
		Username: config.Username,
		Password: config.Token,
	}
	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:   repo.Url,
		Depth: GitDepth,
		Auth:  auth,
		Bare:  false,
	})
	if err != nil {
		return nil, nil, err
	}

	return r, fs, nil
}

func (o *Operations) IsValidForBoneClone(repo *git.Repository, config domain.Config) (bool, domain.RemoteConfig, error) {
	rCfg := domain.RemoteConfig{}

	headRef, err := repo.Head()
	if err != nil {
		return false, rCfg, err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return false, rCfg, err
	}
	tree, err := headCommit.Tree()
	if err != nil {
		return false, rCfg, err
	}

	file, err := tree.File(config.Identifier.Filename)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return false, rCfg, nil
		}
		return false, rCfg, err
	}

	blob, err := repo.BlobObject(file.Hash)
	if err != nil {
		return false, rCfg, err
	}

	reader, err := blob.Reader()
	if err != nil {
		return false, rCfg, err
	}
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	if err != nil {
		return false, rCfg, err
	}

	k := koanf.NewWithConf(koanf.Conf{Delim: ".", StrictMerge: true})
	if err := k.Load(rawbytes.Provider(content), yaml.Parser()); err != nil {
		// Invalid YAML -> skip repository without error
		return false, domain.RemoteConfig{}, nil
	}
	if err := k.Unmarshal("", &rCfg); err != nil {
		// Invalid structure -> skip repository without error
		return false, domain.RemoteConfig{}, nil
	}

	skel := strings.TrimSpace(config.Identifier.Name)
	if skel == "" {
		return false, rCfg, nil
	}
	for _, a := range rCfg.Accepts {
		if strings.TrimSpace(a) == skel {
			return true, rCfg, nil
		}
	}
	return false, rCfg, nil
}

func (o *Operations) CopyFiles(
	repo *git.Repository,
	fs billy.Filesystem,
	config domain.Config,
	provider domain.ProviderConfig,
	targetBranch string,
) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	// Ensure we are operating on the desired target branch (if provided)
	if err := ensureOnTargetBranch(repo, worktree, targetBranch); err != nil {
		return err
	}

	for _, definedFile := range config.Files.Include {
		files, err := getAllFilenames(definedFile)
		if err != nil {
			return err
		}

		for _, file := range files {
			if isExcluded(file, config.Files.Exclude) {
				continue
			}
			if err := writeAndStageFile(fs, worktree, file); err != nil {
				return err
			}
		}

		upToDate, err := commitAndPush(repo, worktree, config, provider, targetBranch)
		if err != nil {
			return err
		}
		if upToDate {
			return nil
		}
	}

	return nil
}

// Wrapper functions for backward compatibility with existing callers.
func CloneGit(repo domain.GitRepository, config domain.ProviderConfig) (*git.Repository, billy.Filesystem, error) {
	return DefaultOps.CloneGit(repo, config)
}

func IsValidForBoneClone(repo *git.Repository, config domain.Config) (bool, domain.RemoteConfig, error) {
	return DefaultOps.IsValidForBoneClone(repo, config)
}

func CopyFiles(
	repo *git.Repository,
	fs billy.Filesystem,
	config domain.Config,
	provider domain.ProviderConfig,
	targetBranch string,
) error {
	return DefaultOps.CopyFiles(repo, fs, config, provider, targetBranch)
}

func writeAndStageFile(fs billy.Filesystem, worktree *git.Worktree, file string) error {
	// Ensure directory exists
	parts := strings.Split(file, "/")
	if len(parts) > 1 {
		dir := strings.Join(parts[:len(parts)-1], "/")
		if _, err := fs.Lstat(dir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if err := fs.MkdirAll(dir, 0o755); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	f, err := fs.Create(file)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if _, err = f.Write(content); err != nil {
		return err
	}
	if _, err = worktree.Add(file); err != nil {
		return err
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

// ensureOnTargetBranch ensures the worktree is on the provided target branch.
// If the branch doesn't exist locally, it tries origin/<branch>, otherwise bases
// the new branch on the current HEAD.
func ensureOnTargetBranch(repo *git.Repository, worktree *git.Worktree, targetBranch string) error {
	tb := strings.TrimSpace(targetBranch)
	if tb == "" {
		return nil
	}

	tbRef := plumbing.NewBranchReferenceName(tb)
	// Attempt to detect current HEAD; if this fails we'll rely on checkout errors
	headRef, _ := repo.Head()
	if headRef != nil && headRef.Name() == tbRef {
		return nil
	}

	// Try to checkout the branch if it already exists locally
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: tbRef}); err == nil {
		return nil
	}

	// Create the branch, basing it off origin/<tb> when available, else current HEAD
	var baseHash plumbing.Hash
	if rref, rerr := repo.Reference(plumbing.NewRemoteReferenceName("origin", tb), true); rerr == nil {
		baseHash = rref.Hash()
	} else if headRef != nil {
		baseHash = headRef.Hash()
	}

	co := &git.CheckoutOptions{Branch: tbRef, Create: true}
	if baseHash != (plumbing.Hash{}) {
		co.Hash = baseHash
	}
	return worktree.Checkout(co)
}

// commitAndPush creates a commit with configured author defaults and pushes it.
// It returns alreadyUpToDate=true when the push indicates no changes.
func commitAndPush(repo *git.Repository, worktree *git.Worktree, config domain.Config, provider domain.ProviderConfig, targetBranch string) (bool, error) {
	name := config.Git.Name
	if name == "" {
		name = DefaultCommitterName
	}
	email := config.Git.Email
	if email == "" {
		email = DefaultCommitterEmail
	}

	if _, err := worktree.Commit("Updated via boneclone", &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
	}); err != nil {
		return false, err
	}

	opts := &git.PushOptions{
		Auth: &http.BasicAuth{Username: provider.Username, Password: provider.Token},
	}
	if tb := strings.TrimSpace(targetBranch); tb != "" {
		localRef := "refs/heads/" + tb
		opts.RefSpecs = []gogitcfg.RefSpec{gogitcfg.RefSpec(localRef + ":" + localRef)}
	}
	if err := repo.Push(opts); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return true, nil
		}
		return false, err
	}
	return false, nil
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
