# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build the manager binary
# --platform=$BUILDPLATFORM pins the builder stage to the machine doing the build, so a
# multi-platform build cross-compiles with the Go toolchain instead of running the whole stage
# under QEMU emulation for every target.
FROM --platform=$BUILDPLATFORM golang:1.26.3 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY pkg/ pkg/

# Build
# TARGETARCH is supplied per platform by buildx, and MUST be declared with no default. Giving a
# predefined platform argument a default makes BuildKit treat it as an ordinary build argument and
# use that default, so `ARG TARGETARCH=amd64` reported amd64 even when building for arm64 -- which
# put an amd64 binary inside the arm64 manifest, the exact failure this was meant to prevent. The
# shell default below covers a plain `docker build` (make docker-build, used by the e2e suites),
# where the variable may be unset.
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-$(go env GOARCH)} GO111MODULE=on go build -a -o adapter main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /tmp
COPY --from=builder --chown=nonroot:nonroot /workspace/adapter /adapter
USER 65532:65532

ENTRYPOINT ["/adapter"]
