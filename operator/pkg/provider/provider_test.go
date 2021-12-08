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

package provider

import (
	"testing"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func Test_paramValue_extractValue(t *testing.T) {
	type args struct {
		requirements map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "golden path",
			args: args{requirements: map[string]string{
				"service.str.0":  "v1",
				"service.byte.1": "7c",
				"service.str.2":  "productpage",
				"service.byte.3": "7c",
				"service.str.4":  "bookinfo",
				"service.byte.5": "7c",
				"service.str.6":  "demo",
				"service.byte.7": "7c",
				"service.byte.8": "2d",
			}},
			want: "v1|productpage|bookinfo|demo|-",
		},
		{
			name: "empty",
			args: args{requirements: map[string]string{
				"instance.str.100": "productpage",
				"instance.str.0":   "v1",
				"instance.byte.49": "7c",
			}},
			want: "",
		},
		{
			name: "random",
			args: args{requirements: map[string]string{
				"service.str.100": "productpage",
				"service.str.0":   "v1",
				"service.byte.49": "7c",
			}},
			want: "v1|productpage",
		},
		{
			name: "gap",
			args: args{requirements: map[string]string{
				"service.str.0":   "v1",
				"service.byte.50": "7c",
				"service.str.110": "productpage",
			}},
			want: "v1|productpage",
		},
		{
			name: "invalid_byte",
			args: args{requirements: map[string]string{
				"service.byte.50": "invalid",
				"service.str.110": "productpage",
			}},
			wantErr: true,
		},
		{
			name: "invalid_key",
			args: args{requirements: map[string]string{
				"service.byte": "7c",
			}},
			wantErr: true,
		},
		{
			name: "invalid_key_index",
			args: args{requirements: map[string]string{
				"service.byte.ab": "7c",
			}},
			wantErr: true,
		},
		{
			name: "single_entity",
			args: args{requirements: map[string]string{
				"service": "productpage",
			}},
			want: "productpage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := &paramValue{
				key: "service",
				val: nil,
			}
			requirements := make([]labels.Requirement, len(tt.args.requirements))
			for key, v := range tt.args.requirements {
				r, _ := labels.NewRequirement(key, selection.In, []string{v})
				requirements = append(requirements, *r)
			}
			if err := pv.extractValue(requirements); (err != nil) != tt.wantErr {
				t.Errorf("extractValue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && *pv.val != tt.want {
				t.Errorf("extractValue() val = %v, want %v", *pv.val, tt.want)
			}
		})
	}
}
