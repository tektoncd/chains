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
	"errors"
	"os"
)

// FromEnvironment tries to build a client-only config from environment variables. If none are set then returns nil.
func FromEnvironment() (*Config, error) {
	remoteURL := os.Getenv("RELIC_URL")
	if remoteURL == "" {
		return nil, nil
	}
	clientCert := os.Getenv("RELIC_CLIENT_CERT")
	clientKey := os.Getenv("RELIC_CLIENT_KEY")
	if clientCert == "" {
		return nil, errors.New("RELIC_CLIENT_CERT must be set when RELIC_URL is set")
	}
	if clientKey == "" {
		clientKey = clientCert
	}
	cfg := &Config{
		Remote: &RemoteConfig{
			DirectoryURL: remoteURL,
			KeyFile:      clientKey,
			CertFile:     clientCert,
			CaCert:       os.Getenv("RELIC_CACERT"),
		},
	}
	return cfg, cfg.Normalize("<env>")
}
