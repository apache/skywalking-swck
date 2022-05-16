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

package manifests

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

var _ kubernetes.Repo = &AssetsRepo{}

//go:embed fetcher injector oapserver satellite storage ui
var manifests embed.FS

// AssetsRepo provides templates through assets
type AssetsRepo struct {
	Root string
}

func NewRepo(component string) *AssetsRepo {
	return &AssetsRepo{Root: component}
}

// ReadFile reads the content of compiled in files at path and returns a buffer with the data.
func (a *AssetsRepo) ReadFile(path string) ([]byte, error) {
	return manifests.ReadFile(path)
}

func (a *AssetsRepo) GetFilesRecursive(path string) ([]string, error) {
	absolutePath := fmt.Sprintf("%s/%s", a.Root, path)

	rootFI, err := Stat(absolutePath)
	if err != nil {
		return nil, err
	}

	return getFilesRecursive(filepath.Dir(absolutePath), rootFI)
}

func getFilesRecursive(prefix string, root os.FileInfo) ([]string, error) {
	if !root.IsDir() {
		return nil, fmt.Errorf("not a dir: %s", root.Name())
	}
	prefix = filepath.Join(prefix, root.Name())
	fs, _ := manifests.ReadDir(prefix)
	out := make([]string, 0)
	for _, f := range fs {
		info, err := Stat(filepath.Join(prefix, f.Name()))
		if err != nil {
			return nil, err
		}
		if !f.IsDir() {
			out = append(out, filepath.Join(prefix, f.Name()))
			continue
		}
		nfs, err := getFilesRecursive(prefix, info)
		if err != nil {
			return nil, err
		}
		out = append(out, nfs...)
	}
	return out, nil
}

// Stat returns a FileInfo object for the given path.
func Stat(path string) (os.FileInfo, error) {
	f, err := manifests.Open(path)
	if err != nil {
		return nil, err
	}

	return f.Stat()
}
