/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Version returns a cobra command to be added to another cobra command, like:
// ```go
//	rootCmd.AddCommand(version.Version())
// ```
func Version() *cobra.Command {
	var outputJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := GetVersionInfo()
			v.Name = cmd.Root().Name()
			v.Description = cmd.Root().Short
			if outputJSON {
				out, err := v.JSONString()
				if err != nil {
					return errors.Wrap(err, "unable to generate JSON from version info")
				}
				cmd.Println(out)
			} else {
				cmd.Println(v.String())
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "print JSON instead of text")

	return cmd
}
