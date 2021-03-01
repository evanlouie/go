package helm

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// BuildInfo encapsulates the parsed output of running `helm template`.
type BuildInfo struct {
	Version      string
	GitCommit    string
	GitTreeState string
	GoVersion    string
}

// Version runs `helm version` and parses the output.
func Version() (v BuildInfo, err error) {
	// Run `helm version` and capture the output
	cmd := exec.Command("helm", "version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return v, fmt.Errorf(`running %s: %s: %w`, cmd, stderr.String(), err)
	}
	if stderr.String() != "" {
		return v, fmt.Errorf(`running %s: %s`, cmd, stderr.String())
	}

	// capture against stdout
	rgx := regexp.MustCompile(`(?i)Version:"(?P<Version>v\d+\.\d+\.\d+)".*GitCommit:"(?P<GitCommit>[^"]+)".*GitTreeState:"(?P<GitTreeState>[^"]+)".*GoVersion:"(?P<GoVersion>[^"]+)"`)
	matchNames := rgx.SubexpNames()
	for idx, matchValue := range rgx.FindStringSubmatch(stdout.String()) {
		switch matchNames[idx] {
		case "Version":
			v.Version = matchValue
		case "GitCommit":
			v.GitCommit = matchValue
		case "GitTreeState":
			v.GitTreeState = matchValue
		case "GoVersion":
			v.GoVersion = matchValue
		}
	}

	return v, err
}

// parse a semantic version the output of the BuildInfo outputted from
// `helm template` into struct of ints.
func (v BuildInfo) parse() (parsed struct{ major, minor, fix int }, err error) {
	// regex capture group names
	const (
		major = "Major"
		minor = "Minor"
		fix   = "Fix"
	)
	// build the regex string and compile
	rgxStr := fmt.Sprintf(`(?i)v(?P<%s>\d+)\.(?P<%s>\d+)\.(?P<%s>\d+)`, major, minor, fix)
	semVerRgx := regexp.MustCompile(rgxStr)

	// iterate over captures and assign values to parsed
	for idx, value := range semVerRgx.FindStringSubmatch(v.Version) {
		captureName := semVerRgx.SubexpNames()[idx]
		var valueAsInt int
		// atoi the value if it is in a named capture group ("Major", "Minor", "Fix")
		// WARN should never reach error case unless additional capture groups are
		// added to the regex which try to capture non-atoi-able strings.
		if captureName != "" {
			var err error
			valueAsInt, err = strconv.Atoi(value)
			if err != nil {
				return parsed, fmt.Errorf(`parsing version string %s as int: %w`, value, err)
			}
		}

		// assign based on the capture group name
		switch captureName {
		case major:
			parsed.major = valueAsInt
		case minor:
			parsed.minor = valueAsInt
		case fix:
			parsed.fix = valueAsInt
		}
	}

	return parsed, nil
}

// IsHelm3 determines if the provided Helm version is major version 3.
func (v BuildInfo) IsHelm3() bool {
	return strings.HasPrefix(strings.ToLower(v.Version), "v3.")
}

// IsHelm2 determines if the provided Helm version is major version 2.
func (v BuildInfo) IsHelm2() bool {
	return strings.HasPrefix(strings.ToLower(v.Version), "v2.")
}
