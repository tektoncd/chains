package config

import (
	"context"
	"sync/atomic"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	Artifacts    ArtifactConfigs
	Storage      StorageConfigs
	Signers      SignerConfigs
	Builder      BuilderConfig
	Transparency TransparencyConfig
	SPIRE        SPIREConfig
}

// ArtifactConfig contains the configuration for how to sign/store/format the signatures for each artifact type
type ArtifactConfigs struct {
	TaskRuns Artifact
	OCI      Artifact
}

// Artifact contains the configuration for how to sign/store/format the signatures for a single artifact
type Artifact struct {
	Format         string
	StorageBackend string
	Signer         string
}

// StorageConfig contains the configuration to instantiate different storage providers
type StorageConfigs struct {
	GCS    GCSStorageConfig
	OCI    OCIStorageConfig
	Tekton TektonStorageConfig
	DocDB  DocDBStorageConfig
}

// SigningConfig contains the configuration to instantiate different signers
type SignerConfigs struct {
	PGP  PGPSigner
	X509 X509Signer
	KMS  KMSSigner
}

type BuilderConfig struct {
	ID string
}

type PGPSigner struct {
}

type X509Signer struct {
}

type KMSSigner struct {
	KMSRef string
}

type GCSStorageConfig struct {
	Bucket string
}

type OCIStorageConfig struct {
	Repository string
	Insecure   bool
}

type TektonStorageConfig struct {
}

type DocDBStorageConfig struct {
	URL string
}

type TransparencyConfig struct {
	Enabled bool
	URL     string
}

type SPIREConfig struct {
	Enabled bool
}

const (
	taskrunFormatKey  = "artifacts.taskrun.format"
	taskrunStorageKey = "artifacts.taskrun.storage"
	taskrunSignerKey  = "artifacts.taskrun.signer"

	ociFormatKey  = "artifacts.oci.format"
	ociStorageKey = "artifacts.oci.storage"
	ociSignerKey  = "artifacts.oci.signer"

	gcsBucketKey             = "storage.gcs.bucket"
	ociRepositoryKey         = "storage.oci.repository"
	ociRepositoryInsecureKey = "storage.oci.repository.insecure"
	docDBUrlKey              = "storage.docdb.url"
	// No config needed for Tekton object storage

	// No config needed for pgp signer
	// No config needed for x509 signer

	// KMS
	kmsSignerKMSRef = "signers.kms.kmsref"

	// Builder config
	builderIDKey = "builder.id"

	transparencyEnabledKey = "transparency.enabled"
	transparencyURLKey     = "transparency.url"

	// SPIRE config
	spireEnabledKey = "spire.enabled"

	chainsConfig = "chains-config"
)

var defaults = map[string]string{
	taskrunFormatKey:   "tekton",
	taskrunStorageKey:  "tekton",
	taskrunSignerKey:   "x509",
	ociFormatKey:       "simplesigning",
	ociStorageKey:      "oci",
	ociSignerKey:       "x509",
	transparencyURLKey: "https://rekor.sigstore.dev",

	builderIDKey: "tekton-chains",
}

var supportedValues = map[string][]string{
	taskrunFormatKey:  {"tekton", "in-toto"},
	taskrunStorageKey: {"tekton", "oci", "gcs", "docdb"},
	taskrunSignerKey:  {"pgp", "x509", "kms"},
	ociFormatKey:      {"tekton", "simplesigning"},
	ociStorageKey:     {"tekton", "oci", "gcs", "docdb"},
	ociSignerKey:      {"pgp", "x509", "kms"},
}

func parse(data map[string]string, logger *zap.SugaredLogger) Config {
	cfg := Config{}

	// Artifact-specific configs

	// TaskRuns
	cfg.Artifacts.TaskRuns.Format = valueOrDefault(taskrunFormatKey, data, logger)
	cfg.Artifacts.TaskRuns.StorageBackend = valueOrDefault(taskrunStorageKey, data, logger)
	cfg.Artifacts.TaskRuns.Signer = valueOrDefault(taskrunSignerKey, data, logger)

	// OCI
	cfg.Artifacts.OCI.Format = valueOrDefault(ociFormatKey, data, logger)
	cfg.Artifacts.OCI.StorageBackend = valueOrDefault(ociStorageKey, data, logger)
	cfg.Artifacts.OCI.Signer = valueOrDefault(ociSignerKey, data, logger)

	// Storage level configs

	cfg.Storage.GCS.Bucket = valueOrDefault(gcsBucketKey, data, logger)
	cfg.Storage.OCI.Repository = valueOrDefault(ociRepositoryKey, data, logger)
	cfg.Storage.OCI.Insecure = (valueOrDefault(ociRepositoryInsecureKey, data, logger) == "true")
	cfg.Storage.DocDB.URL = valueOrDefault(docDBUrlKey, data, logger)

	cfg.Transparency.Enabled = (valueOrDefault(transparencyEnabledKey, data, logger) == "true")
	cfg.Transparency.URL = valueOrDefault(transparencyURLKey, data, logger)

	cfg.SPIRE.Enabled = (valueOrDefault(spireEnabledKey, data, logger) == "true")

	cfg.Signers.KMS.KMSRef = valueOrDefault(kmsSignerKMSRef, data, logger)

	// Build config
	cfg.Builder.ID = valueOrDefault(builderIDKey, data, logger)

	return cfg
}

func valueOrDefault(key string, data map[string]string, logger *zap.SugaredLogger) string {
	if v, ok := data[key]; ok {
		if validate(key, v) {
			return v
		} else {
			logger.Warnf("[%s] is not a valid option for key [%s], using default [%s] instead. please set [%s] to one of %v in the config\n", v, key, defaults[key], key, supportedValues[key])
		}
	}
	return defaults[key]
}

func validate(key, value string) bool {
	vals, ok := supportedValues[key]
	// if it doesn't exist in supportedValues, we don't validate
	if !ok {
		return true
	}
	for _, v := range vals {
		if v == value {
			return true
		}
	}
	return false
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
			config := parse(cm.Data, cs.logger)
			// Swap the values!
			cs.config.Store(config)
			cs.logger.Infof("config store %s updated: %v", cs.name, cm.Data)
		}
	}()
}

// NewConfigStore returns a store that is configured to watch the configmap for changes.
func NewConfigStore(kc kubernetes.Interface, namespace string, logger *zap.SugaredLogger) (*ConfigStore, error) {
	opts := metav1.SingleObject(metav1.ObjectMeta{Name: chainsConfig})
	w, err := kc.CoreV1().ConfigMaps(namespace).Watch(context.TODO(), opts)
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
