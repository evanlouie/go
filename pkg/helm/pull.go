package helm

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
)

// Pull will do a `helm pull` for the target chart and extract the chart to
// `into`.
// If an existing repository is found in in the host helm client with same
// repository URL, the chart will be pulled from that repository instead of
// using the "--repo" option.
// Note that the directory structure will look like: <into>/<chart>/Chart.yaml
func Pull(repoURL string, chart string, version string, into string) error {
	// check if existing repo with same URL in host client
	existingRepo, err := FindRepoNameByURL(repoURL)
	if err != nil {
		return err
	}
	if existingRepo != "" {
		chart = path.Join(existingRepo, chart) // set chart to the form of <repo_name>/<path_to_chart>
		repoURL = ""                           // zero out so --repo is not used
	}

	// arguments don't include --repo by default
	pullArgs := []string{
		"pull", chart,
		"--untar",          // untar
		"--untardir", into, // untar into the target directory instead of cwd
	}

	// provide a --version if specified
	if version != "" {
		pullArgs = append(pullArgs, "--version", version)
	}

	// use the --repo option to pull directly from URL if repo not on host Helm
	if repoURL != "" {
		pullArgs = append(pullArgs, "--repo", repoURL)
	}

	cmd := exec.Command("helm", pullArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", err, stderr.String())
	}

	return nil
}
