//
// Copyright (c) SAS Institute Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package config

import (
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/sassoftware/relic/lib/certloader"
	"gopkg.in/yaml.v2"
)

const (
	defaultSigXchg = "relic.signatures"
	sigKey         = "relic.signatures"
)

var (
	Version   = "unknown" // set this at link time
	Commit    = "unknown" // set this at link time
	UserAgent = "relic/" + Version
	Author    = "SAS Institute Inc."
)

type TokenConfig struct {
	Type       string  // Provider type: file or pkcs11 (default)
	Provider   string  // Path to PKCS#11 provider module (required)
	Label      string  // Select a token by label
	Serial     string  // Select a token by serial number
	Pin        *string // PIN to use, otherwise will be prompted. Can be empty. (optional)
	Timeout    int     // (server) Terminate command after N seconds (default 60)
	Retries    int     // (server) Retry failed commands N times (default 5)
	User       *uint   // User argument for PKCS#11 login (optional)
	UseKeyring bool    // Read PIN from system keyring

	name string
}

type KeyConfig struct {
	Token           string   // Token section to use for this key (linux)
	Alias           string   // This is an alias for another key
	Label           string   // Select a key by label
	ID              string   // Select a key by ID (hex notation)
	PgpCertificate  string   // Path to PGP certificate associated with this key
	X509Certificate string   // Path to X.509 certificate associated with this key
	KeyFile         string   // For "file" tokens, path to the private key
	Roles           []string // List of user roles that can use this key
	Timestamp       bool     // If true, attach a timestamped countersignature when possible
	Hide            bool     // If true, then omit this key from 'remote list-keys'

	name  string
	token *TokenConfig
}

type ServerConfig struct {
	Listen     string // Port to listen for TLS connections
	ListenHTTP string // Port to listen for plaintext connections
	KeyFile    string // Path to TLS key file
	CertFile   string // Path to TLS certificate chain
	LogFile    string // Optional error log

	Disabled    bool // Always return 503 Service Unavailable
	ListenDebug bool // Serve debug info on an alternate port
	NumWorkers  int  // Number of worker subprocesses per configured token

	TokenCheckInterval int
	TokenCheckFailures int
	TokenCheckTimeout  int

	// URLs to all servers in the cluster. If a client uses DirectoryURL to
	// point to this server (or a load balancer), then we will give them these
	// URLs as a means to distribute load without needing a middle-box.
	Siblings []string
}

type ClientConfig struct {
	Nickname    string   // Name that appears in audit log entries
	Roles       []string // List of roles that this client possesses
	Certificate string   // Optional CA certificate(s) that sign client certs instead of using fingerprint-based auth

	certs *x509.CertPool
}

type RemoteConfig struct {
	URL            string `yaml:",omitempty"` // URL of remote server
	DirectoryURL   string `yaml:",omitempty"` // URL of directory server
	KeyFile        string `yaml:",omitempty"` // Path to TLS client key file
	CertFile       string `yaml:",omitempty"` // Path to TLS client certificate
	CaCert         string `yaml:",omitempty"` // Path to CA certificate
	ConnectTimeout int    `yaml:",omitempty"` // Connection timeout in seconds
	Retries        int    `yaml:",omitempty"` // Attempt an operation (at least) N times
}

type TimestampConfig struct {
	URLs      []string // List of timestamp server URLs
	MsURLs    []string // List of microsoft-style URLs
	Timeout   int      // Connect timeout in seconds
	CaCert    string   // Path to CA certificate
	Memcache  []string // host:port of memcached to use for caching timestamps
	RateLimit float64  // limit timestamp requests per second
	RateBurst int      // allow burst of requests before limit kicks in
}

type AmqpConfig struct {
	URL      string // AMQP URL to report signatures to i.e. amqp://user:password@host
	CaCert   string
	KeyFile  string
	CertFile string
	SigsXchg string // Name of exchange to send to (default relic.signatures)
}

type Config struct {
	Tokens    map[string]*TokenConfig  `yaml:",omitempty"`
	Keys      map[string]*KeyConfig    `yaml:",omitempty"`
	Server    *ServerConfig            `yaml:",omitempty"`
	Clients   map[string]*ClientConfig `yaml:",omitempty"`
	Remote    *RemoteConfig            `yaml:",omitempty"`
	Timestamp *TimestampConfig         `yaml:",omitempty"`
	Amqp      *AmqpConfig              `yaml:",omitempty"`

	PinFile string `yaml:",omitempty"` // Optional YAML file with additional token PINs

	path string
}

func ReadFile(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := new(Config)
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, config.Normalize(path)
}

func (config *Config) Normalize(path string) error {
	config.path = path
	normalized := make(map[string]*ClientConfig)
	for fingerprint, client := range config.Clients {
		if client.Certificate != "" {
			certs, err := certloader.ParseX509Certificates([]byte(client.Certificate))
			if err != nil {
				return fmt.Errorf("invalid certificate for client %s: %s", fingerprint, err)
			}
			client.certs = x509.NewCertPool()
			for _, cert := range certs {
				client.certs.AddCert(cert)
			}
		} else if len(fingerprint) != 64 {
			return errors.New("Client keys must be hex-encoded SHA256 digests of the public key")
		}
		lower := strings.ToLower(fingerprint)
		normalized[lower] = client
	}
	config.Clients = normalized
	if config.PinFile != "" {
		contents, err := ioutil.ReadFile(config.PinFile)
		if err != nil {
			return fmt.Errorf("error reading PinFile: %s", err)
		}
		pinMap := make(map[string]string)
		if err := yaml.Unmarshal(contents, pinMap); err != nil {
			return fmt.Errorf("error reading PinFile: %s", err)
		}
		for token, pin := range pinMap {
			tokenConf := config.Tokens[token]
			if tokenConf != nil {
				ppin := pin
				tokenConf.Pin = &ppin
			}
		}
	}
	for tokenName, tokenConf := range config.Tokens {
		tokenConf.name = tokenName
		if tokenConf.Type == "" {
			tokenConf.Type = "pkcs11"
		}
	}
	for keyName, keyConf := range config.Keys {
		keyConf.name = keyName
		if keyConf.Token != "" {
			keyConf.token = config.Tokens[keyConf.Token]
		}
	}
	if s := config.Server; s != nil {
		if s.TokenCheckInterval == 0 {
			s.TokenCheckInterval = 60
		}
		if s.TokenCheckTimeout == 0 {
			s.TokenCheckTimeout = 60
		}
		if s.TokenCheckFailures == 0 {
			s.TokenCheckFailures = 3
		}
	}
	if r := config.Remote; r != nil {
		if r.ConnectTimeout == 0 {
			r.ConnectTimeout = 15
		}
		if r.Retries == 0 {
			r.Retries = 3
		}
	}
	return nil
}

func (config *Config) GetToken(tokenName string) (*TokenConfig, error) {
	if config.Tokens == nil {
		return nil, errors.New("No tokens defined in configuration")
	}
	tokenConf, ok := config.Tokens[tokenName]
	if !ok {
		return nil, fmt.Errorf("Token \"%s\" not found in configuration", tokenName)
	}
	return tokenConf, nil
}

func (config *Config) NewToken(name string) *TokenConfig {
	if config.Tokens == nil {
		config.Tokens = make(map[string]*TokenConfig)
	}
	tok := &TokenConfig{name: name}
	config.Tokens[name] = tok
	return tok
}

func (config *Config) GetKey(keyName string) (*KeyConfig, error) {
	keyConf, ok := config.Keys[keyName]
	if !ok {
		return nil, fmt.Errorf("Key \"%s\" not found in configuration", keyName)
	} else if keyConf.Alias != "" {
		keyConf, ok = config.Keys[keyConf.Alias]
		if !ok {
			return nil, fmt.Errorf("Alias \"%s\" points to undefined key \"%s\"", keyName, keyConf.Alias)
		}
	}
	if keyConf.Token == "" {
		return nil, fmt.Errorf("Key \"%s\" does not specify required value 'token'", keyName)
	}
	return keyConf, nil
}

func (config *Config) NewKey(name string) *KeyConfig {
	if config.Keys == nil {
		config.Keys = make(map[string]*KeyConfig)
	}
	key := &KeyConfig{name: name}
	config.Keys[name] = key
	return key
}

func (config *Config) Path() string {
	return config.path
}

func (config *Config) GetTimestampConfig() (*TimestampConfig, error) {
	tconf := config.Timestamp
	if tconf == nil {
		return nil, errors.New("No timestamp section exists in the configuration")
	}
	return tconf, nil
}

// ListServedTokens returns a list of token names that are accessible by at least one role
func (config *Config) ListServedTokens() []string {
	names := make(map[string]bool)
	for _, key := range config.Keys {
		if len(key.Roles) != 0 {
			names[key.Token] = true
		}
	}
	ret := make([]string, 0, len(names))
	for name := range names {
		ret = append(ret, name)
	}
	return ret
}

func (tconf *TokenConfig) Name() string {
	return tconf.name
}

func (aconf *AmqpConfig) ExchangeName() string {
	if aconf.SigsXchg != "" {
		return aconf.SigsXchg
	}
	return defaultSigXchg
}

func (aconf *AmqpConfig) RoutingKey() string {
	return sigKey
}
