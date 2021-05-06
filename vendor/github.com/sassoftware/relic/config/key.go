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

import "time"

const defaultTimeout = 60 * time.Second

func (keyConf *KeyConfig) Name() string {
	return keyConf.name
}

func (keyConf *KeyConfig) GetTimeout() time.Duration {
	if keyConf.token != nil && keyConf.token.Timeout != 0 {
		return time.Second * time.Duration(keyConf.token.Timeout)
	}
	return defaultTimeout
}

func (keyConf *KeyConfig) SetToken(tokenConf *TokenConfig) {
	keyConf.Token = tokenConf.name
	keyConf.token = tokenConf
}
