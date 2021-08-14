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

package injector

import (
	"testing"
)

func TestValidateBool(t *testing.T) {
	type args struct {
		annotation string
		value      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test true",
			args: args{
				annotation: "test annotation",
				value:      "true",
			},
			wantErr: false,
		},
		{
			name: "test false",
			args: args{
				annotation: "test annotation",
				value:      "false",
			},
			wantErr: false,
		},
		{
			name: "test no",
			args: args{
				annotation: "test annotation",
				value:      "no",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateBool(tt.args.annotation, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("ValidateBool() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateInt(t *testing.T) {
	type args struct {
		annotation string
		value      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test 1",
			args: args{
				annotation: "test annotation",
				value:      "1",
			},
			wantErr: false,
		},
		{
			name: "test -1",
			args: args{
				annotation: "test annotation",
				value:      "-1",
			},
			wantErr: false,
		},
		{
			name: "test other",
			args: args{
				annotation: "test annotation",
				value:      "true",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateInt(tt.args.annotation, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateClassCacheMode(t *testing.T) {
	type args struct {
		annotation string
		value      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test MEMORY",
			args: args{
				annotation: "test annotation",
				value:      "MEMORY",
			},
			wantErr: false,
		},
		{
			name: "test FILE",
			args: args{
				annotation: "test annotation",
				value:      "FILE",
			},
			wantErr: false,
		},
		{
			name: "test CACHE",
			args: args{
				annotation: "test annotation",
				value:      "CACHE",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateClassCacheMode(tt.args.annotation, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("ValidateClassCacheMode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIpandPort(t *testing.T) {
	type args struct {
		annotation string
		value      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid ip and valid port",
			args: args{
				annotation: "test annotation",
				value:      "127.0.0.1:8080",
			},
			wantErr: false,
		},
		{
			name: "valid ip and valid port",
			args: args{
				annotation: "test annotation",
				value:      "localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "invalid ip and valid port",
			args: args{
				annotation: "test annotation",
				value:      "192.0.1.288:8080",
			},
			wantErr: true,
		},
		{
			name: "valid ip and invalid port",
			args: args{
				annotation: "test annotation",
				value:      "192.0.1.255:88888",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateIpandPort(tt.args.annotation, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("ValidateIpandPort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLoggingLevel(t *testing.T) {
	type args struct {
		annotation string
		value      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test TRACE",
			args: args{
				annotation: "test annotation",
				value:      "TRACE",
			},
			wantErr: false,
		},
		{
			name: "test DEBUG",
			args: args{
				annotation: "test annotation",
				value:      "DEBUG",
			},
			wantErr: false,
		},
		{
			name: "test INFO",
			args: args{
				annotation: "test annotation",
				value:      "INFO",
			},
			wantErr: false,
		},
		{
			name: "test WARN",
			args: args{
				annotation: "test annotation",
				value:      "WARN",
			},
			wantErr: false,
		},
		{
			name: "test ERROR",
			args: args{
				annotation: "test annotation",
				value:      "ERROR",
			},
			wantErr: false,
		},
		{
			name: "test OFF",
			args: args{
				annotation: "test annotation",
				value:      "OFF",
			},
			wantErr: false,
		},
		{
			name: "test other",
			args: args{
				annotation: "test annotation",
				value:      "other",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateLoggingLevel(tt.args.annotation, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("ValidateLoggingLevel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
