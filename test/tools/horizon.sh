#!/usr/bin/env bash

#
# Licensed to Apache Software Foundation (ASF) under one or more contributor
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
#

# Query the Horizon UI's BFF the way a person does: log in, then call an API
# with the session cookie. Modelled on apache/skywalking-helm's
# test/e2e/script/horizon.sh.
#
#   horizon.sh <base-url> get  <api-path>
#   horizon.sh <base-url> post <api-path> <json-body>
#
# Horizon protects /api/* with RBAC and ships `auth.local.users` empty, so a UI
# deployed with the operator's generated horizon.yaml has no way in at all --
# `/api/auth/health` reports configured: false and every login is refused. The
# e2e resources that use this seed a user through the UI resource's spec.config;
# a failure at the login step here means that seeding did not take effect, which
# is the signal, not a flake.
#
# Credentials come from HORIZON_USERNAME / HORIZON_PASSWORD and default to the
# admin/admin pair those resources seed.

set -eo pipefail

BASE_URL=$1
VERB=$2
API_PATH=$3
BODY=${4:-}

USERNAME=${HORIZON_USERNAME:-admin}
PASSWORD=${HORIZON_PASSWORD:-admin}

JAR=$(mktemp)
trap 'rm -f "$JAR"' EXIT

curl -sS --fail-with-body -c "$JAR" -X POST "${BASE_URL}/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}" >/dev/null

grep -q 'horizon_sid' "$JAR" || { echo "login did not set a session cookie" >&2; exit 1; }

if [ "$VERB" = "post" ]; then
  curl -sS --fail-with-body -b "$JAR" -X POST "${BASE_URL}${API_PATH}" \
    -H 'Content-Type: application/json' -d "$BODY"
else
  curl -sS --fail-with-body -b "$JAR" "${BASE_URL}${API_PATH}"
fi
