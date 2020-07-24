package config

import (
	"strings"
	"sync/atomic"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	Artifacts Artifacts
}

type Artifacts struct {
	TaskRuns TaskRuns
}

type TaskRuns struct {
	Formats         Formats
	StorageBackends StorageBackends
}

type Formats struct {
	EnabledFormats map[string]struct{}
}

type StorageBackends struct {
	EnabledBackends map[string]struct{}
}

const (
	taskrunEnabledFormatsKey  = "artifacts.taskrun.formats.enabled"
	taskrunEnabledStoragesKey = "artifacts.taskrun.storage.enabled"
)

func parse(data map[string]string) Config {
	cfg := Config{}

	// Start with artifact-specific configs
	// TaskRuns
	cfg.Artifacts.TaskRuns.Formats.EnabledFormats = stringSet(taskrunEnabledFormatsKey, data)
	cfg.Artifacts.TaskRuns.StorageBackends.EnabledBackends = stringSet(taskrunEnabledStoragesKey, data)

	return cfg
}

func stringSet(key string, data map[string]string) map[string]struct{} {
	var names []string
	if val := data[key]; val != "" {
		// We have to check if the key is set before passing to strings.Split
		// strings.Split on an empty string results in a slice like: []string{""}, when we
		// really want an empty slice
		names = strings.Split(val, ",")
	}
	result := map[string]struct{}{}
	for _, name := range names {
		result[name] = struct{}{}
	}
	return result
}

type ConfigStore struct {
	name   string
	config atomic.Value

	c      <-chan watch.Event
	logger *zap.SugaredLogger
}

func (cs *ConfigStore) Config() Config {
	return cs.config.Load().(Config)
}

func (cs *ConfigStore) watch() {
	go func() {
		for evt := range cs.c {
			cm := evt.Object.(*corev1.ConfigMap)
			cs.logger.Debugf("watch event %s on %s/%s", evt.Type, cm.Namespace, cm.Name)
			config := parse(cm.Data)
			// Swap the values!
			cs.config.Store(config)
			cs.logger.Infof("config store %s updated", cs.name)
		}
	}()
}

// NewConfigStore returns a store that is configured to watch the configmap for changes.
func NewConfigStore(configStore string, kc kubernetes.Interface, namespace, name string, logger *zap.SugaredLogger) (*ConfigStore, error) {
	opts := metav1.SingleObject(metav1.ObjectMeta{Name: name})
	w, err := kc.CoreV1().ConfigMaps(namespace).Watch(opts)
	if err != nil {
		return nil, err
	}
	val := atomic.Value{}
	val.Store(Config{})
	cs := ConfigStore{
		name:   configStore,
		c:      w.ResultChan(),
		config: val,
		logger: logger,
	}
	cs.logger.Debug("staring watch on configmap: %s/%s", namespace, name)
	cs.watch()
	return &cs, nil
}
