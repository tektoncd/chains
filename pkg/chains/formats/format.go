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

package formats

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/spire"
	"go.uber.org/zap"
	"knative.dev/pkg/apis"
)

// Payloader is an interface to generate a chains Payload from a TaskRun
type Payloader interface {
	CreatePayload(ctx context.Context, obj interface{}) (interface{}, error)
	Type() PayloadType
	Wrap() bool
}

type PayloadType string

// If you update this, remember to update AllFormatters
const (
	PayloadTypeTekton        PayloadType = "tekton"
	PayloadTypeSimpleSigning PayloadType = "simplesigning"
	PayloadTypeInTotoIte6    PayloadType = "in-toto"
)

var AllFormatters = []PayloadType{PayloadTypeTekton, PayloadTypeSimpleSigning, PayloadTypeInTotoIte6}

func VerifySpire(ctx context.Context, tr *v1beta1.TaskRun, spireControllerAPI *spire.SpireControllerApiClient, logger *zap.SugaredLogger) error {
	if err := verifySignedTaskrrunResults(tr); err != nil {
		return err
	} else {
		if len(tr.Status.TaskRunResults) > 0 {
			logger.Info("spire taskrun status condition verified")
		}
	}
	if err := spireControllerAPI.VerifyStatusInternalAnnotation(ctx, tr, logger); err != nil {
		return errors.Wrap(err, "verifying SPIRE")
	} else {
		logger.Info("internal status annotation verified by spire")
	}
	return nil
}

func verifySignedTaskrrunResults(tr *v1beta1.TaskRun) error {
	if len(tr.Status.TaskRunResults) > 0 {
		taskRunCondition := tr.Status.GetCondition(apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String()))
		if taskRunCondition != nil {
			if taskRunCondition.IsFalse() {
				return errors.New("taskrun status condition not verified. Spire taskrun results verification failure")
			}
		} else {
			return fmt.Errorf("could not find condition Type %s in taskrun status", v1beta1.TaskRunConditionResultsVerified.String())
		}
	}
	return nil
}
