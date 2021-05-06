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

package passprompt

import (
	"fmt"
	"io"
)

type LoginFunc func(string) (bool, error)

func Login(login LoginFunc, getter PasswordGetter, keyringService, keyringUser, initialPrompt, failPrefix string) error {
	keyringFirst := keyringService != ""
	prompt := initialPrompt
	for {
		var password string
		var err error
		if keyringFirst {
			keyringFirst = false
			password, err = keyringGet(keyringService, keyringUser)
			if err == errNotFound {
				continue
			} else if err != nil {
				return fmt.Errorf("keyring error: %s", err)
			}
		} else if getter != nil {
			password, err = getter.GetPasswd(prompt)
			if err != nil {
				return err
			} else if password == "" {
				return io.EOF
			}
		} else {
			return io.EOF
		}
		ok, err := login(password)
		if err != nil {
			return err
		} else if ok {
			if keyringService != "" {
				err := keyringSet(keyringService, keyringUser, password)
				if err != nil {
					return fmt.Errorf("keyring error: %s", err)
				}
			}
			return nil
		}
		prompt = failPrefix + initialPrompt
	}
}
