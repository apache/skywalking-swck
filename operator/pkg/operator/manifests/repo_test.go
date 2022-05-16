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
	"testing"
)

func TestStat(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantName  string
		wantIsDir bool
		wantErr   bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:      "current directory",
			path:      ".",
			wantName:  ".",
			wantIsDir: true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Stat(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if got.Name() != tt.wantName {
					t.Errorf("Stat() got.Name() = %v, want %v", got, tt.wantName)
				}
				if got.IsDir() != tt.wantIsDir {
					t.Errorf("Stat() got.IsDir() = %v, want %v", got, tt.wantIsDir)
				}
			}
		})
	}
}

func TestAssetsRepo_ReadFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "read a template file",
			path:    "oapserver/templates/deployment.yaml",
			wantErr: false,
		},
		{
			name:    "read a non-existent file",
			path:    "non-existent/foo.yaml",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AssetsRepo{
				Root: "",
			}
			_, err := a.ReadFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestAssetsRepo_GetFilesRecursive(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		path      string
		wantFiles bool // Since filenames may be changed, only check if the number of files is greater than 0.
		wantErr   bool
	}{
		{
			name:      "get oap template files",
			root:      "oapserver",
			path:      "templates",
			wantFiles: true,
			wantErr:   false,
		},
		{
			name:      "get ui template files",
			root:      "ui",
			path:      "templates",
			wantFiles: true,
			wantErr:   false,
		},
		{
			name:      "get files from ui directory without path",
			root:      "ui",
			path:      "",
			wantFiles: false,
			wantErr:   true,
		},
		{
			name:      "get files in a non-existent directory",
			root:      "non-existent",
			path:      "templates",
			wantFiles: false,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AssetsRepo{
				Root: tt.root,
			}
			got, err := a.GetFilesRecursive(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFilesRecursive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantFiles && len(got) <= 0 {
				t.Errorf("GetFilesRecursive() should return more than one filenames")
			}
		})
	}
}
