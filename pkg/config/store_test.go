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
	name := "my-config"
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	fakekubeclient := fakek8s.NewSimpleClientset(cm)

	cs, err := NewConfigStore("test-store", fakekubeclient, ns, name, logtesting.TestLogger(t))
	if err != nil {
		t.Errorf("error creating new config store: %v", err)
	}

	// Test that there's nothing with an empty configmap
	if len(cs.Config().EnabledFormats) != 0 {
		t.Errorf("unexpected data: %v", cs.Config().EnabledFormats)
	}

	// Setup some config
	cm.Data = map[string]string{}
	cm.Data[enabledFormatsKey] = "foo,bar"

	if cm, err = fakekubeclient.CoreV1().ConfigMaps(ns).Update(cm); err != nil {
		t.Errorf("error updating configmap: %v", err)
	}

	// It should be updated by then...
	time.Sleep(100 * time.Millisecond)
	// Test that the values are set!
	if diff := cmp.Diff([]string{"foo", "bar"}, cs.Config().EnabledFormats); diff != "" {
		t.Error(diff)
	}

	// Change it again
	cm.Data[enabledFormatsKey] = "foo,bar,baz"

	if _, err := fakekubeclient.CoreV1().ConfigMaps(ns).Update(cm); err != nil {
		t.Errorf("error updating configmap: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	// Test that the values are set!
	if diff := cmp.Diff([]string{"foo", "bar", "baz"}, cs.Config().EnabledFormats); diff != "" {
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
				EnabledFormats: nil,
			},
		},
		{
			name: "single",
			data: map[string]string{enabledFormatsKey: "foo"},
			want: Config{
				EnabledFormats: []string{"foo"},
			},
		},
		{
			name: "multiple",
			data: map[string]string{enabledFormatsKey: "foo,bar"},
			want: Config{
				EnabledFormats: []string{"foo", "bar"},
			},
		},
		{
			name: "extra",
			data: map[string]string{enabledFormatsKey: "foo,bar", "other-key": "foo"},
			want: Config{
				EnabledFormats: []string{"foo", "bar"},
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
