/*
Copyright 2024 The Tekton Authors

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

package taskrunmetrics

import (
	"context"

	"k8s.io/client-go/rest"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterClient(func(ctx context.Context, _ *rest.Config) context.Context { return WithClient(ctx) })
}

// RecorderKey is used for associating the Recorder inside the context.Context.
type RecorderKey struct{}

// WithClient adds a metrics recorder to the given context
func WithClient(ctx context.Context) context.Context {
	rec, err := NewRecorder(ctx)
	if err != nil {
		logging.FromContext(ctx).Errorf("Failed to create taskrun metrics recorder %v", err)
	}
	return context.WithValue(ctx, RecorderKey{}, rec)
}

// Get extracts the taskrunmetrics.Recorder from the context.
func Get(ctx context.Context) *Recorder {
	untyped := ctx.Value(RecorderKey{})
	if untyped == nil {
		logging.FromContext(ctx).Panic("Unable to fetch *taskrunmetrics.Recorder from context.")
	}
	return untyped.(*Recorder)
}
