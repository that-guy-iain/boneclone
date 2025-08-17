BoneClone Developer Guidelines

Audience: Advanced Go developers contributing to BoneClone.

Overview
- Purpose: Keep downstream repositories in sync with a skeleton template by discovering repositories across multiple providers (GitHub, GitLab, Azure DevOps), cloning shallow copies, validating an identifier marker, copying selected files from the local working directory into each repo, committing, and pushing.
- Key components:
  - CLI (main.go) using urfave/cli v3 and koanf for config.
  - Providers in app/infra/git/repository_providers (GitHub, GitLab, Azure) implementing domain.GitRepositoryProvider.
  - Git ops in app/infra/git/operations.go implemented with go-git v6 and go-billy memfs.

Build and Configuration
- Go toolchain: go 1.24 (see go.mod).
- Module path: go.iain.rocks/boneclone
- Build:
  - Local build: go build -o boneclone .
  - Install into GOPATH/bin: go install go.iain.rocks/boneclone@latest
- Run:
  - boneclone -c path/to/config.yaml
  - Default config path flag is --config (alias -c) and defaults to .boneclone.yaml if not provided.
- Configuration schema (koanf YAML):
  providers: array of provider entries
    - provider: github | gitlab | azure
      username: string (used for HTTP BasicAuth username in pushes/clones)
      org: string
        - GitHub/GitLab: organization/group name
        - Azure DevOps: organization URL, e.g. "https://dev.azure.com/example/"
      token: string (Personal Access Token for provider API and for git HTTP BasicAuth password)
  files:
    include: [string]  # files or directories relative to the current working directory
    exclude: [string]  # file paths to skip (exact match against the discovered file list)
  identifier:
    filename: string   # a file that must exist in the target repository
    content: string    # substring that must appear in the file contents

- Notes:
  - main.go loads config with koanf (StrictMerge=true). It currently does not expand environment variables in values. A helper expandEnvValues exists but is not invoked; if you need ${VAR} expansion, call it before Unmarshal.
  - The copy step reads from the local filesystem (os.ReadFile) for each included path and writes to an in-memory filesystem (memfs) tied to the cloned repo. Ensure you run the CLI from the skeleton template root so include paths resolve correctly.
  - Shallow clones: operations.CloneGit uses Depth=1. If you need history-dependent behavior, increase GIT_DEPTH.

Testing
- Running tests:
  - Full suite: go test ./...
  - With race detector: go test -race ./...
  - Specific package: go test ./app/infra/git/repository_providers -run 'GitlabProvider'
- External dependencies:
  - Tests are designed to run offline; provider tests inject fakes and do not hit network APIs.
  - Azure: repository_providers/azure.go exposes constructor vars newCoreClient and newGitClient, allowing tests to inject fakes (see azure_test.go). Avoid creating real azuredevops.Connection in tests.
  - GitLab: gitlab.go defines a narrow interface gitlabGroupProjectLister used by GitLab provider; tests stub Groups.ListGroupProjects and control paging (see gitlab_test.go). Observe NextPage pagination behavior.
  - GitHub: github.go currently uses the real github.Client. To add unit tests without network, prefer introducing a small interface for the subset used (Repositories.ListByOrg) or a constructor var similar to Azure to inject a fake.
- Git operations:
  - operations.IsValidForBoneClone: checks that identifier.filename exists in the HEAD tree and that the blob contains identifier.content (substring match, case-sensitive). It uses go-git to read tree/file contents.
  - operations.CopyFiles: for each include path, it collects files recursively (getAllFilenames), filters via isExcluded (exact string match), writes file content from the local workspace into memfs, stages, commits ("Updated via boneclone"), then pushes with BasicAuth. It returns early with nil if push reports go-git.NoErrAlreadyUpToDate.
  - Avoid invoking CopyFiles in unit tests unless you stub push. There is no seam for Push yet; if you need to unit test, consider extracting Push into an overridable function or checking Worktree state pre-push.
- Adding new tests:
  - Prefer small interfaces over concrete SDK clients to enable fakes (pattern used by GitLab and Azure providers).
  - Keep tests hermetic by using t.TempDir for any filesystem interactions. For helpers like getAllFilenames and isExcluded, follow operations_test.go patterns.
  - Example (validated locally):
    // app/infra/git/example_demo_test.go
    package git
    import "testing"
    func TestIsExcluded_Demo(t *testing.T) {
        if isExcluded("foo/bar.txt", []string{"baz.txt"}) {
            t.Fatalf("expected foo/bar.txt not to be excluded")
        }
    }
    # Run: go test ./app/infra/git -run Demo
    # This pattern was verified on 2025-08-17 by temporarily adding the file, running go test (passed), then removing it.
- Coverage:
  - go test ./... -cover -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html

Additional Development Information
- There should be no "magic values" where a check against a value is done against a literal in the codebase; all values should be defined as constants.
- All code must be ran through the linter and all errors fixed: golangci-lint run ./... 
- Concurrency: main.go spins a goroutine per repository and uses a sync.WaitGroup to wait. There is currently no rate limiting or backoff for provider APIs or pushes; if you observe throttling, introduce a worker pool or limiter.
- Error handling: main.go currently prints errors to stdout and continues; consider structured logging and per-repo error aggregation for large runs.
- Provider factory: repository_providers.NewProvider selects by strings.ToLower(provider). Unknown providers return an error.
- Authentication:
  - Clone/push use HTTP BasicAuth with Username=config.Username and Password=config.Token. Some providers ignore the username but require a non-empty value ("x-access-token" is a common placeholder for GitHub).
- Cross-platform paths: getAllFilenames uses os.Lstat and walks directories. Exclusions are exact string matches against the discovered file list (with OS-specific separators). Use consistent path formatting in config to match the runtime OS.
- Identifiers: The identifier check uses substring search rather than YAML/JSON parsing. Be precise with content to avoid false positives/negatives.
- Future testability improvements (recommended):
  - Introduce a git remote/push interface in operations.go so Push can be stubbed in tests.
  - Add an interface around GitHub client similar to GitLab/Azure patterns.
  - Wire expandEnvValues into config load for secrets via environment variables.

Quickstart
1) Build: go build -o boneclone .
2) Configure: copy example/multi-providers.yaml to .boneclone.yaml and edit tokens/orgs (for Azure, org is the full URL).
3) Dry run: consider commenting out the Push call in operations.CopyFiles while testing changes, or run against a disposable repo.
4) Test: go test -race ./...

Git Usage:
- ALL CHANGES ARE TO BE STAGED TO GIT
- DO NOT COMMIT DIRECTLY TO THE REPOSITORY