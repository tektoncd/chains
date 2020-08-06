package config

import (
	"sync/atomic"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	Artifacts Artifacts
	Storage   Storage
}

type Artifacts struct {
	TaskRuns Artifact
	OCI      Artifact
}

type Artifact struct {
	Format         string
	StorageBackend string
}

type Storage struct {
	GCS GCS
}

type GCS struct {
	Bucket string
}

const (
	taskrunFormatKey  = "artifacts.taskrun.format"
	taskrunStorageKey = "artifacts.taskrun.storage"
	gcsBucketKey      = "storage.gcs.bucket"

	ociFormatKey  = "artifacts.oci.format"
	ociStorageKey = "artifacts.oci.storage"

	chainsConfig = "chains-config"
)

func parse(data map[string]string) Config {
	cfg := Config{}

	// Start with artifact-specific configs
	// TaskRuns
	cfg.Artifacts.TaskRuns.Format = data[taskrunFormatKey]
	cfg.Artifacts.TaskRuns.StorageBackend = data[taskrunStorageKey]

	// Storage level configs
	cfg.Storage.GCS.Bucket = data[gcsBucketKey]
	// OCI
	cfg.Artifacts.OCI.Format = data[ociFormatKey]
	cfg.Artifacts.OCI.StorageBackend = data[ociStorageKey]

	return cfg
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
			cs.logger.Infof("config store %s updated: %v", cs.name, cm.Data)
		}
	}()
}

// NewConfigStore returns a store that is configured to watch the configmap for changes.
func NewConfigStore(kc kubernetes.Interface, namespace string, logger *zap.SugaredLogger) (*ConfigStore, error) {
	opts := metav1.SingleObject(metav1.ObjectMeta{Name: chainsConfig})
	w, err := kc.CoreV1().ConfigMaps(namespace).Watch(opts)
	if err != nil {
		return nil, err
	}
	val := atomic.Value{}
	val.Store(Config{})
	cs := ConfigStore{
		name:   chainsConfig,
		c:      w.ResultChan(),
		config: val,
		logger: logger,
	}
	cs.logger.Debug("staring watch on configmap: %s/%s", namespace, chainsConfig)
	cs.watch()
	return &cs, nil
}
