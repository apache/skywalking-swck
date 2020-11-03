// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package build

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	l "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const yamlHeader = `# Licensed to Apache Software Foundation (ASF) under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. Apache Software Foundation (ASF) licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

`

func NewLicense() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "Apply license header to generated manifests",
	}
	cmd.AddCommand(newInsert())
	cmd.AddCommand(newCheck())
	return cmd
}

func newInsert() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "insert",
		Short: "Insert license header to generated manifests",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) < 1 || args[0] == "" {
				return fmt.Errorf("lack the path in args")
			}
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get present directory: %w", err)
			}
			w := filepath.Join(dir, args[0])
			l.Info("working directory: ", w)

			err = filepath.Walk(w,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					l.Debugf("path: %s", path)
					if info.IsDir() {
						return nil
					}
					insertHeader(path)
					return nil
				})
			if err != nil {
				return fmt.Errorf("failed to process files: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func newCheck() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check yaml files licenses",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) < 1 || args[0] == "" {
				return fmt.Errorf("lack the path in args")
			}
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get present directory: %w", err)
			}
			w := filepath.Join(dir, args[0])
			l.Info("working directory: ", w)

			err = filepath.Walk(w,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					l.Debugf("path: %s", path)
					if info.IsDir() {
						return nil
					}
					data, err := ioutil.ReadFile(path)
					if err != nil {
						l.Errorf("failed to read data from file %s: %v", path, err)
					}
					if checkHeader(data) {
						return nil
					}
					return fmt.Errorf("failed to check %s", path)
				})
			if err != nil {
				return fmt.Errorf("failed to process files: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func insertHeader(path string) {
	l.Debugf("============Processing file %s================", path)
	var buff bytes.Buffer
	data, err := ioutil.ReadFile(path)
	if err != nil {
		l.Errorf("failed to read data from file %s: %v", path, err)
	}
	if checkHeader(data) {
		return
	}

	if _, err = buff.WriteString(yamlHeader); err != nil {
		l.Errorf("failed to write header to %s: %v", path, err)
	}
	if _, err = buff.Write(append(data, '\n')); err != nil {
		l.Errorf("failed to write content to %s: %v", path, err)
	}
	if err = ioutil.WriteFile(path, buff.Bytes(), os.ModePerm); err != nil {
		l.Errorf("failed to update %s: %v", path, err)
	}
	l.Info("updated ", path)
}

func checkHeader(data []byte) bool {
	return strings.Contains(string(data), "Apache Software Foundation (ASF)")
}
