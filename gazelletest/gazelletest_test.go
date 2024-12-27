package gazelletest

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestGazelleLoadsBuildFileConsistently(t *testing.T) {
	// 1. Determine the Bazel root directory from the environment variable.
	gitRepoRoot, ok := os.LookupEnv("SOURCE_REPO_PATH")
	if !ok {
		t.Fatalf("SOURCE_REPO_PATH environment variable not set. This test must be run within a Bazel environment.")
	}

	{
		gitDiffsDetected, statusText, err := hasGitDiffs(gitRepoRoot)
		if err != nil {
			t.Fatalf("error running git diff in %s: %v", gitRepoRoot, err)
		}
		if gitDiffsDetected {
			t.Fatalf("this test must run in a git directory with no diffs; status:\n%s", statusText)
		}
	}

	// 2. Define the path to the BUILD.bazel file.
	buildFilePath := path.Join(gitRepoRoot, "proto", "example", "BUILD.bazel")

	loadBuildFileContent := func() string {
		content, err := os.ReadFile(buildFilePath)
		if err != nil {
			t.Fatalf("failed to read BUILD.bazel file: %v", err)
		}
		return string(content)

	}

	runGazelle := func(context string) {
		// 4. Run `bazel run //gazelle` for the first time.
		cmd := exec.Command("bazel", "run", "//:gazelle")
		cmd.Dir = gitRepoRoot
		cmd.Stdout = os.Stdout // Pipe output to standard output for visibility
		cmd.Stderr = os.Stderr // Pipe errors to standard error for visibility
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s: gazelle execution failed: %v", context, err)
		}
	}

	initialContent := loadBuildFileContent()
	runGazelle("first run")
	firstRunContent := loadBuildFileContent()
	runGazelle("second run")
	secondRunContent := loadBuildFileContent()
	t.Logf("Initial content of %s:\n%s", buildFilePath, initialContent)
	t.Logf("Diff after first gazelle run:\n%s", unifiedDiff(initialContent, firstRunContent))
	t.Logf("Diff after second gazelle run:\n%s", unifiedDiff(firstRunContent, secondRunContent))

	// 8. Assert that the file content is the same after both Gazelle executions.
	if diff := cmp.Diff(string(firstRunContent), string(secondRunContent)); diff != "" {
		t.Errorf("BUILD.bazel file content differs after two Gazelle runs (-first +second):\n%s", diff)
		t.Errorf("Initial BUILD.bazel file content:\n%s", string(initialContent))
		t.Errorf("First run BUILD.bazel file content:\n%s", string(firstRunContent))
		t.Errorf("Second run BUILD.bazel file content:\n%s", string(secondRunContent))
	}
}

func unifiedDiff(a, b string) string {
	dmp := diffmatchpatch.New()

	// Compute the diffs
	diffs := dmp.DiffMain(a, b, false)

	// Generate the unified diff
	return dmp.DiffPrettyText(diffs)
}

// hasGitDiffs checks if there are any uncommitted changes (diffs) in the
// Git workspace of the given directory.
func hasGitDiffs(repoDir string) (bool, string, error) {
	// Use "git status --porcelain" to get a concise summary of changes.
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		// Check if the error is because the directory is not a Git repository.
		if exitError, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitError.Stderr), "not a git repository") {
				return false, "", fmt.Errorf("not a git repository: %v", err)
			}
		}
		return false, "", fmt.Errorf("failed to execute git status: %v", err)
	}

	// If the output is empty, there are no changes.
	return len(output) > 0, string(output), nil
}
