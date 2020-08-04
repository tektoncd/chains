package config

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestNewConfigStore(t *testing.T) {

	ns := "my-namespace"
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "chains-config",
			Namespace: ns,
		},
	}
	fakekubeclient := fakek8s.NewSimpleClientset(cm)

	cs, err := NewConfigStore(fakekubeclient, ns, logtesting.TestLogger(t))
	if err != nil {
		t.Errorf("error creating new config store: %v", err)
	}

	// Test that there's nothing with an empty configmap
	if cs.Config().Artifacts.TaskRuns.Format != "" {
		t.Errorf("unexpected data: %v", cs.Config().Artifacts.TaskRuns.Format)
	}

	// Setup some config
	cm.Data = map[string]string{}
	cm.Data[taskrunFormatKey] = "foo"

	if cm, err = fakekubeclient.CoreV1().ConfigMaps(ns).Update(cm); err != nil {
		t.Errorf("error updating configmap: %v", err)
	}

	// It should be updated by then...
	time.Sleep(100 * time.Millisecond)
	// Test that the values are set!
	if diff := cmp.Diff("foo", cs.Config().Artifacts.TaskRuns.Format); diff != "" {
		t.Error(diff)
	}

	// Change it again
	cm.Data[taskrunFormatKey] = "bar"

	if _, err := fakekubeclient.CoreV1().ConfigMaps(ns).Update(cm); err != nil {
		t.Errorf("error updating configmap: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	// Test that the values are set!
	if diff := cmp.Diff("bar", cs.Config().Artifacts.TaskRuns.Format); diff != "" {
		t.Error(diff)
	}
}

func Test_parse(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string
		want Config
	}{
		{
			name: "empty",
			data: map[string]string{},
			want: Config{
				Artifacts: Artifacts{
					TaskRuns: TaskRuns{},
				},
			},
		},
		{
			name: "single",
			data: map[string]string{taskrunFormatKey: "foo"},
			want: Config{
				Artifacts: Artifacts{
					TaskRuns: TaskRuns{
						Format:         "foo",
						StorageBackend: "",
					},
				},
			},
		},
		{
			name: "extra",
			data: map[string]string{taskrunFormatKey: "foo", "other-key": "foo"},
			want: Config{
				Artifacts: Artifacts{
					TaskRuns: TaskRuns{
						Format:         "foo",
						StorageBackend: "",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parse(tt.data)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("parse() = %v", diff)
			}
		})
	}
}
