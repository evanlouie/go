package helm

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	yamlPlus "github.com/evanlouie/go/pkg/yaml"
	"gopkg.in/yaml.v3"
)

// TemplateOptions encapsulate the options for `helm template`.
// helm template \
//   --repo <Repo> \
//   --version <Version> \
//   --namespace <Namespace> --create-namespace \
//   --values <Values[0]> --values <Value[1]> ... \
//   --set <Set[0]> --set <Set[1]> ... \
//   <Release> <Chart>
type TemplateOptions struct {
	Release   string   // [NAME]
	Chart     string   // [CHART]
	Repo      string   // --repo
	Version   string   // --version
	Namespace string   // --namespace flag. implies --create-namespace
	Values    []string // "--value" flags. e.g.: ["foo/bar.yaml", "/etc/my/values.yaml"] == "--values foo/bar.yaml -- values /et/my/values.yaml"
	Set       []string // "--set" flags. e.g: ["foo=bar", "baz=123"] == "--set foo=bar --set baz=123"
}

// TemplateWithCRDs will `helm template` the target chart as well as ensure
// that any YAML files in the the charts "crds" directory are prepended to
// the returned YAML string -- which are not templated via "helm template" in
// helm 3.
//
// Starting with Helm 3, the "crds" directory of a chart holds a special meaning
// and holds CRD YAMLs which are not templated -- thus not outputted from
// `helm template` -- but installed to the cluster via `helm install`. This
// function is useful to get a complete YAML output for the entire chart.
func TemplateWithCRDs(opts TemplateOptions) ([]map[string]interface{}, error) {
	// interpertet the chart path based on if a repo-url was provided
	var chartPath, crdPath string
	if opts.Repo != "" {
		tmpDir, err := os.MkdirTemp("", "fabrikate")
		if err != nil {
			return nil, fmt.Errorf(`creating temporary directory to pull helm chart %s@%s from %s: %w`, opts.Chart, opts.Version, opts.Repo, err)
		}
		defer os.RemoveAll(tmpDir)
		if err := Pull(opts.Repo, opts.Chart, opts.Version, tmpDir); err != nil {
			return nil, fmt.Errorf(`pulling helm chart %s@%s from %s: %w`, opts.Chart, opts.Version, opts.Repo, err)
		}
		chartPath = filepath.Join(tmpDir, opts.Chart)
	} else {
		chartPath = opts.Chart
	}
	crdPath = filepath.Join(chartPath, "crds")

	// walk the "crds" dir to collect all the yaml strings
	var crds []string // list of crd yaml <strings>
	if info, err := os.Stat(crdPath); err == nil {
		if info.IsDir() {
			err := filepath.Walk(crdPath, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf(`walking path %s: %w`, path, err)
				}
				extension := strings.ToLower(filepath.Ext(info.Name()))
				// track all yaml files
				if !info.IsDir() && extension == ".yaml" {
					crd, err := os.ReadFile(path)
					if err != nil {
						return fmt.Errorf("reading CRD file %s: %w", path, err)
					}
					crds = append(crds, string(crd))
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf(`walking CRD path %s: %w`, crdPath, err)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(`reading helm chart CRD directory %s: %w`, crdPath, err)
	}

	// run `helm template` to get the contents of the pulled chart
	templateOpts := opts           // inherit all the initial settings
	templateOpts.Repo = ""         // zero out so it wont attempt to lookup the repo
	templateOpts.Chart = chartPath // manually set the path of the chart to the downloaded chart
	template, err := Template(templateOpts)
	if err != nil {
		return nil, fmt.Errorf(`templating helm chart at %s: %w`, templateOpts.Chart, err)
	}

	// join all the yaml together with "---"
	allYAMLEntries := append(crds, template)
	unifiedYAMLString := strings.TrimSpace(strings.Join(allYAMLEntries, "\n---\n"))

	// convert to maps and remove all nils
	var maps, noNils []map[string]interface{}
	maps, err = yamlPlus.DecodeMaps([]byte(unifiedYAMLString))
	if err != nil {
		return nil, fmt.Errorf(`parsing output of "helm template": %w`, err)
	}
	for _, m := range maps {
		if m != nil {
			noNils = append(noNils, m)
		}
	}

	return noNils, nil
}

// Template runs `helm template` on the chart specified by opts.
// Returns the string output of stdout for `helm template`.
// Will have a non-nil error if an error occurs when running the command or the
// command outputs ANYTHING to stdout.
//
// NOTE in Helm 3, CRDs in the "crds" directory of the chart are not outputted
// from `helm template` but are installed via `helm install`
func Template(opts TemplateOptions) (string, error) {
	templateArgs := []string{"template"}
	if opts.Repo != "" {
		// if an existing helm repo exists on the helm client, use that for templating
		existingRepo, err := FindRepoNameByURL(opts.Repo)
		if err != nil {
			return "", fmt.Errorf(`searching existing helm repositories for %s: %w`, opts.Repo, err)
		}
		if existingRepo != "" {
			opts.Chart = existingRepo + "/" + opts.Chart
		} else {
			// if an existing repo is not found, use the --repo option to pull from network
			templateArgs = append(templateArgs, "--repo", opts.Repo)
		}
	}
	if opts.Namespace != "" {
		templateArgs = append(templateArgs, "--create-namespace", "--namespace", opts.Namespace)
	}
	for _, set := range opts.Set {
		templateArgs = append(templateArgs, "--set", set)
	}
	for _, yamlPath := range opts.Values {
		templateArgs = append(templateArgs, "--values", yamlPath)
	}

	// a helm release [NAME] is specified as an optional leading parameter to the [CHART]
	if opts.Release != "" {
		templateArgs = append(templateArgs, opts.Release)
	}
	templateArgs = append(templateArgs, opts.Chart)

	templateCmd := exec.Command("helm", templateArgs...)
	var stdout, stderr bytes.Buffer
	templateCmd.Stdout = &stdout
	templateCmd.Stderr = &stderr

	if err := templateCmd.Run(); err != nil {
		return "", fmt.Errorf(`running "%s": %v: %v`, templateCmd, err, stderr.String())
	}
	if stderr.Len() != 0 {
		return "", fmt.Errorf(`"%s" exited with output to stderr: %s`, templateCmd, stderr.String())
	}

	return stdout.String(), nil
}

func injectNamespace(manifest map[string]interface{}, namespace string) (map[string]interface{}, error) {
	if manifest == nil {
		return nil, nil
	}
	// inject the metadata map if it is not present
	if _, ok := manifest["metadata"]; !ok {
		manifest["metadata"] = map[string]interface{}{}
	}

	metadata, ok := manifest["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(`reflecting metadata of yaml manifest: %+v`, manifest)
	}
	if metadata["namespace"] != nil {
		return nil, fmt.Errorf(`existing namespace found in yaml: %+v`, manifest)
	}
	metadata["namespace"] = namespace

	return manifest, nil
}

func injectNamespaceBack(unifiedManifest string, namespace string) (string, error) {
	manifests, err := yamlPlus.DecodeMaps([]byte(unifiedManifest))
	if err != nil {
		return "", fmt.Errorf(`unmarshalling yaml into []map[string]interface{}: %s: %w`, unifiedManifest, err)
	}

	// add namespace to manifest metdata ONLY if it does not already have one
	var withInjectedNS []string
	for _, manifest := range manifests {
		// nil? create the metadata map
		if manifest["metadata"] == nil {
			manifest["metadata"] = map[string]interface{}{
				"namespace": namespace,
			}
		} else {
			metadata, ok := manifest["metadata"].(map[string]interface{})
			// only add the namespace if the "metadata" is a map and namespace is not set
			if !ok {
				// its not a map[string]interface{}? error!
				return "", fmt.Errorf(`"metadata" of manifest is not a map[string]interface{}: %+v`, manifest)
			} else if ok && metadata["namespace"] == nil || metadata["namespace"] == "" {
				// is it a map[string]interface{} with a zero value for "namespace"? set the namespace
				metadata["namespace"] = namespace
			}
		}

		marshalBytes, err := yaml.Marshal(manifest)
		if err != nil {
			return "", fmt.Errorf(`marshalling yaml for %+v: %w`, manifest, err)
		}
		withInjectedNS = append(withInjectedNS, string(marshalBytes))
	}

	return strings.Join(withInjectedNS, "\n---\n"), nil

	// split the unified manifest string by "---"
	dividerRgx := regexp.MustCompile(`^---$`)
	manifestStrings := dividerRgx.Split(unifiedManifest, -1)

	// parse and inject the namespace into the parsed map
	var injectedManifests []string
	for _, entry := range manifestStrings {
		var m map[interface{}]interface{}
		if err := yaml.Unmarshal([]byte(entry), &m); err != nil {
			return "", fmt.Errorf(`unmarshalling YAML string %s: %w`, entry, err)
		}
		if m["metadata"] != nil {
			metadata, ok := m["metadata"].(map[string]interface{})
			if !ok {
				return "", fmt.Errorf(`reflecting metadata of yaml manifest: %+v`, m)
			}
			if metadata["namespace"] == nil {
				metadata["namespace"] = namespace
			}
		}
		asBytes, err := yaml.Marshal(m)
		if err != nil {
			return "", fmt.Errorf(`marshalling namespace injected YAML %+v: %w`, m, err)
		}
		injectedManifests = append(injectedManifests, string(asBytes))
	}

	// re-join the strings with "---"
	withNS := strings.TrimSpace(strings.Join(injectedManifests, "\n---\n"))

	return strings.TrimSpace(withNS), nil
}

// cleanManifest parses either a yaml document (or list of documents delimitted
// by "---") and removes entries that are not of type map[string]interface{}.
//
// TODO find out if this is needed in helm 3
func cleanManifest(manifest string) (string, error) {
	// split based on yaml divider
	manifests := strings.Split(manifest, "\n---")

	// remove all invalid yaml
	var cleaned []string
	for _, entry := range manifests {
		var m map[string]interface{}
		// if it doesn't unmarshal properly, do not add
		if err := yaml.Unmarshal([]byte(entry), &m); err != nil {
			continue
		}
		// only append documents with a non-empty body
		if len(strings.TrimSpace(entry)) > 0 {
			cleaned = append(cleaned, entry)
		}
	}

	// re-join based on yaml divider
	joined := strings.TrimSpace(strings.Join(cleaned, "\n---"))
	return joined, nil
}

func createNamespace(name string) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]interface{}{
			"name": name,
		},
	}
}
