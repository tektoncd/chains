/*
Copyright 2020 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package artifacts

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// These have the form: sha256:<hex string>
	digestChars = "sh:0123456789abcdef"
)

type Signable interface {
	ExtractObjects(tr *v1beta1.TaskRun) []interface{}
	StorageBackend(cfg config.Config) sets.String
	Signer(cfg config.Config) string
	PayloadFormat(cfg config.Config) formats.PayloadType
	Key(interface{}) string
	Type() string
	Enabled(cfg config.Config) bool
}

type TaskRunArtifact struct {
	Logger *zap.SugaredLogger
}

func (ta *TaskRunArtifact) Key(obj interface{}) string {
	tr := obj.(*v1beta1.TaskRun)
	return "taskrun-" + string(tr.UID)
}

func (ta *TaskRunArtifact) ExtractObjects(tr *v1beta1.TaskRun) []interface{} {
	return []interface{}{tr}
}
func (ta *TaskRunArtifact) Type() string {
	return "tekton"
}

func (ta *TaskRunArtifact) StorageBackend(cfg config.Config) sets.String {
	return cfg.Artifacts.TaskRuns.StorageBackend
}

func (ta *TaskRunArtifact) PayloadFormat(cfg config.Config) formats.PayloadType {
	return formats.PayloadType(cfg.Artifacts.TaskRuns.Format)
}

func (ta *TaskRunArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.TaskRuns.Signer
}

func (ta *TaskRunArtifact) Enabled(cfg config.Config) bool {
	return cfg.Artifacts.TaskRuns.Enabled()
}

type OCIArtifact struct {
	Logger *zap.SugaredLogger
}

type image struct {
	url    string
	digest string
}

// StructuredSignable contains info for signable targets to become either subjects or materials in Intoto statements.
// Name is the identifier used within TaskRun results to identify the target.
// Uri is the resource uri for the target needed iff the target is a material.
// Digest is the target's SHA digest.
// Description describes some basic info of this signable target.
// ProvType determines whether the target is a input (material) or a output (subject). Options are "input" or "output".
type StructuredSignable struct {
	Name        string
	Description string
	Uri         string
	Digest      string
	ProvType    string
}

func (oa *OCIArtifact) ExtractObjects(tr *v1beta1.TaskRun) []interface{} {
	imageResourceNames := map[string]*image{}
	if tr.Status.TaskSpec != nil && tr.Status.TaskSpec.Resources != nil {
		for _, output := range tr.Status.TaskSpec.Resources.Outputs {
			if output.Type == v1beta1.PipelineResourceTypeImage {
				imageResourceNames[output.Name] = &image{}
			}
		}
	}

	for _, rr := range tr.Status.ResourcesResult {
		img, ok := imageResourceNames[rr.ResourceName]
		if !ok {
			continue
		}
		// We have a result for an image!
		if rr.Key == "url" {
			img.url = rr.Value
		} else if rr.Key == "digest" {
			img.digest = rr.Value
		}
	}

	objs := []interface{}{}
	for _, image := range imageResourceNames {
		dgst, err := name.NewDigest(fmt.Sprintf("%s@%s", image.url, image.digest))
		if err != nil {
			oa.Logger.Error(err)
			continue
		}
		objs = append(objs, dgst)
	}

	// Now check TaskResults
	resultImages := ExtractOCIImagesFromResults(tr, oa.Logger)
	objs = append(objs, resultImages...)

	return objs
}

func ExtractOCIImagesFromResults(tr *v1beta1.TaskRun, logger *zap.SugaredLogger) []interface{} {
	taskResultImages := map[string]*image{}
	var objs []interface{}
	urlSuffix := "IMAGE_URL"
	digestSuffix := "IMAGE_DIGEST"
	for _, res := range tr.Status.TaskRunResults {
		if strings.HasSuffix(res.Name, urlSuffix) {
			p := strings.TrimSuffix(res.Name, urlSuffix)
			if v, ok := taskResultImages[p]; ok {
				v.url = strings.Trim(res.Value.StringVal, "\n")
			} else {
				taskResultImages[p] = &image{url: strings.Trim(res.Value.StringVal, "\n")}
			}
		}
		if strings.HasSuffix(res.Name, digestSuffix) {
			p := strings.TrimSuffix(res.Name, digestSuffix)
			if v, ok := taskResultImages[p]; ok {
				v.digest = strings.Trim(res.Value.StringVal, "\n")
			} else {
				taskResultImages[p] = &image{digest: strings.Trim(res.Value.StringVal, "\n")}
			}
		}
	}
	// Only add it if we got both the URL and digest.
	for _, img := range taskResultImages {
		if img != nil && img.url != "" && img.digest != "" {
			dgst, err := name.NewDigest(fmt.Sprintf("%s@%s", img.url, img.digest))
			if err != nil {
				logger.Errorf("error getting digest: %v", err)
				return nil
			}
			objs = append(objs, dgst)
		}
	}

	// look for a comma separated list of images
	for _, key := range tr.Status.TaskRunResults {
		if key.Name != "IMAGES" {
			continue
		}
		imgs := strings.FieldsFunc(key.Value.StringVal, split)

		for _, img := range imgs {
			trimmed := strings.TrimSpace(img)
			if trimmed == "" {
				continue
			}
			dgst, err := name.NewDigest(trimmed)
			if err != nil {
				logger.Errorf("error getting digest for img %s: %v", trimmed, err)
				continue
			}
			objs = append(objs, dgst)
		}
	}

	return objs
}

// ExtractMavenPackagesFromResults extracts Maven packages within TaskRun results and store them as StructuredSignable.
func ExtractMavenPackagesFromResults(tr *v1beta1.TaskRun, logger *zap.SugaredLogger) []interface{} {
	mavenPackages := map[string]*StructuredSignable{}
	var objs []interface{}
	// suffises stores the package, POM and source suffises of a Maven package.
	// Structure is {name_of_target_suffix, digest_suffix, prefix_marker}.
	suffises := [][]string{{"MAVEN_PKG", "MAVEN_PKG_DIGEST", "PKG_"}, {"MAVEN_POM", "MAVEN_POM_DIGEST", "POM_"}, {"MAVEN_SRC", "MAVEN_SRC_DIGEST", "SRC_"}}

	for _, res := range tr.Status.TaskRunResults {
		for _, s := range suffises {
			nameSuffix := s[0]
			digestSuffix := s[1]
			markerPrefix := s[2]
			if strings.HasSuffix(res.Name, nameSuffix) {
				p := strings.TrimSuffix(res.Name, nameSuffix)
				marker := markerPrefix + p
				if v, ok := mavenPackages[marker]; ok {
					v.Name = strings.Trim(res.Value, "\n")
				} else {
					mavenPackages[marker] = &StructuredSignable{Name: strings.Trim(res.Value, "\n")}
				}
				// TODO: add logic for adding Maven package as input.
				mavenPackages[marker].ProvType = "output"
			}
			if strings.HasSuffix(res.Name, digestSuffix) {
				p := strings.TrimSuffix(res.Name, digestSuffix)
				marker := markerPrefix + p
				if v, ok := mavenPackages[marker]; ok {
					v.Digest = strings.Trim(res.Value, "\n")
				} else {
					mavenPackages[marker] = &StructuredSignable{Digest: strings.Trim(res.Value, "\n")}
				}
				mavenPackages[marker].ProvType = "output"
			}

		}
	}
	// Only add it if we got both the package name and digest.
	// TODO: add logic for adding Maven package as input.
	for _, mvn := range mavenPackages {
		if mvn.ProvType == "output" && mvn != nil && mvn.Name != "" && mvn.Digest != "" {
			err := checkDigest(mvn.Digest)
			if err != nil {
				logger.Errorf("error getting digest: %v", err)
			} else {
				objs = append(objs, mvn)
			}
		}
	}

	return objs
}

func checkDigest(name string) error {
	return checkElement("digest", name, digestChars, 7+64, 7+64)
}

// stripRunesFn returns a function which returns -1 (i.e. a value which
// signals deletion in strings.Map) for runes in 'runes', and the rune otherwise.
func stripRunesFn(runes string) func(rune) rune {
	return func(r rune) rune {
		if strings.ContainsRune(runes, r) {
			return -1
		}
		return r
	}
}

// checkElement checks a given named element matches character and length restrictions.
// Returns true if the given element adheres to the given restrictions, false otherwise.
func checkElement(name, element, allowedRunes string, minRunes, maxRunes int) error {
	numRunes := utf8.RuneCountInString(element)
	if (numRunes < minRunes) || (maxRunes < numRunes) {
		return fmt.Errorf("%s must be between %d and %d characters in length: %s", name, minRunes, maxRunes, element)
	} else if len(strings.Map(stripRunesFn(allowedRunes), element)) != 0 {
		return fmt.Errorf("%s can only contain the characters `%s`: %s", name, allowedRunes, element)
	}
	return nil
}

// split allows IMAGES to be separated either by commas (for backwards compatibility)
// or by newlines
func split(r rune) bool {
	return r == '\n' || r == ','
}

func (oa *OCIArtifact) Type() string {
	return "oci"
}

func (oa *OCIArtifact) StorageBackend(cfg config.Config) sets.String {
	return cfg.Artifacts.OCI.StorageBackend
}

func (oa *OCIArtifact) PayloadFormat(cfg config.Config) formats.PayloadType {
	return formats.PayloadType(cfg.Artifacts.OCI.Format)
}

func (oa *OCIArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.OCI.Signer
}

func (oa *OCIArtifact) Key(obj interface{}) string {
	v := obj.(name.Digest)
	return strings.TrimPrefix(v.DigestStr(), "sha256:")[:12]
}

func (oa *OCIArtifact) Enabled(cfg config.Config) bool {
	return cfg.Artifacts.OCI.Enabled()
}
