// +build e2e

/*
Copyright 2021 The Tekton Authors

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

package test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestExamples copies the format in the tektoncd/pipelines repo
// https://github.com/tektoncd/pipeline/blob/main/test/examples_test.go
func TestExamples(t *testing.T) {
	examples := getExamplePaths(t, "../examples/taskruns")
	for _, example := range examples {
		// for each example, make sure the TaskRun executes with the in-toto and taskrun formatters
		runFormatters(t, example)
	}
}

func runFormatters(t *testing.T, example string) {
	// run with the tekton formatter
	//  -- set the correct settings for the formatter
	//  -- run the task and make sure it succeeds
	//  -- verify the annotation on the taskrun
	tektonFormatter(t, example)
	// run with the intoto formatter
	//  -- set the correct settings for the formatter
	//  -- run the task and make sure it succeeds
	//  -- verify the annotation on the taskrun
}

func tektonFormatter(t *testing.T, example string) {
	// no need to set the config, since this is the default
	// run the task
	t.Parallel()
	t.Run(example, func(t *testing.T) {
		t.Logf("creating taskrun %v", example)
		tr := taskRunFromExample(t, example)

		ctx := context.Background()
		c, ns, cleanup := setup(ctx, t, setupOpts{})
		defer cleanup()

		taskRun, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(ctx, tr, metav1.CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		// Give it a minute to complete.
		waitForCondition(ctx, t, c.PipelineClient, taskRun.Name, ns, done, 60*time.Second)
	})
}

func taskRunFromExample(t *testing.T, example string) *v1beta1.TaskRun {
	contents, err := ioutil.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var tr *v1beta1.TaskRun
	if err := yaml.Unmarshal(contents, &tr); err != nil {
		t.Fatal(err)
	}
	return tr
}

func getExamplePaths(t *testing.T, dir string) []string {
	var examplePaths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Fatalf("couldn't walk path %s: %v", path, err)
		}
		// Do not append root and any other folders named "examples"
		if info.Name() == "examples" && info.IsDir() {
			return nil
		}
		if info.IsDir() == false && filepath.Ext(info.Name()) == ".yaml" {
			t.Logf("Adding test %s", path)
			examplePaths = append(examplePaths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("couldn't walk example directory %s: %v", dir, err)
	}
	return examplePaths
}

// func testYamls(t *testing.T, baseDir string, createFunc createFunc, filter pathFilter) {
// 	t.Parallel()
// 	for _, path := range getExamplePaths(t, baseDir, filter) {
// 		path := path // capture range variable
// 		testName := extractTestName(baseDir, path)
// 		waitValidateFunc := waitValidateTaskRunDone
// 		kind := "taskrun"

// 		t.Run(testName, func(t *testing.T) {

// 		})
// 	}
// }

// func extractTestName(baseDir string, path string) string {
// 	re := regexp.MustCompile(baseDir + "/(.+).yaml")
// 	submatch := re.FindSubmatch([]byte(path))
// 	if submatch == nil {
// 		return path
// 	}
// 	return string(submatch[1])
// }
