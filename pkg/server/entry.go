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

package server

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/tektoncd/chains/proto"
	"go.uber.org/zap"
)

func getEntry(uid string, logger *zap.SugaredLogger) (*proto.Entry, error) {
	logger.Infof("Getting entry with uid %s...", uid)
	f := path(uid)
	contents, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, errors.Wrapf(err, "getting entry for %s", uid)
	}
	var e proto.Entry
	if err := json.Unmarshal(contents, &e); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &e, nil
}

func addEntry(e *proto.Entry, logger *zap.SugaredLogger) error {
	logger.Infof("Adding entry %v...", e)
	contents, err := json.Marshal(e)
	if err != nil {
		return errors.Wrap(err, "marshal binary")
	}
	f := path(e.Uid)
	if err := ioutil.WriteFile(f, contents, 0644); err != nil {
		return errors.Wrap(err, "writing file")
	}
	logger.Infof("Successfully added entry at %s...", f)
	return nil
}

func path(uid string) string {
	return filepath.Join(storagePath, uid)
}
