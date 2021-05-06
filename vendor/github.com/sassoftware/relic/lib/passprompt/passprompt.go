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
	"io"
	"os"

	"github.com/howeyc/gopass"
)

// Interface for some means of getting a password from the user (or another source)
type PasswordGetter interface {
	// Print the given prompt and retrieve a password. May return io.EOF if the
	// user cancels the prompt.
	GetPasswd(prompt string) (string, error)
}

// A default password getter using stdin/stderr
type PasswordPrompt struct{}

func (PasswordPrompt) GetPasswd(prompt string) (string, error) {
	passwd, err := gopass.GetPasswdPrompt(prompt, true, os.Stdin, os.Stderr)
	if err == io.EOF {
		return "", nil
	} else if err != nil {
		return "", err
	} else {
		return string(passwd), nil
	}
}
