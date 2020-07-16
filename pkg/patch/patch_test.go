package patch

import (
	"reflect"
	"testing"
)

func TestGetAnnotationsPatch(t *testing.T) {
	tests := []struct {
		name           string
		newAnnotations map[string]string
		want           string
		wantErr        bool
	}{
		{
			name:           "empty",
			newAnnotations: map[string]string{},
			want:           `{"metadata":{}}`,
		},
		{
			name: "one",
			newAnnotations: map[string]string{
				"foo": "bar",
			},
			want: `{"metadata":{"annotations":{"foo":"bar"}}}`,
		},
		{
			name: "many",
			newAnnotations: map[string]string{
				"foo": "bar",
				"baz": "bat",
			},
			want: `{"metadata":{"annotations":{"baz":"bat","foo":"bar"}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAnnotationsPatch(tt.newAnnotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAnnotationsPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotStr := string(got)
			if !reflect.DeepEqual(gotStr, tt.want) {
				t.Errorf("GetAnnotationsPatch() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}
