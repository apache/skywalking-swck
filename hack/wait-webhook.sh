#!/usr/bin/env bash

#
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

TIMEOUT=60

MANIFEST=$(mktemp)

cat <<EOF > $MANIFEST
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServer
metadata:
  name: dummy
spec:
  version: 8.8.1
  instances: 1
  image: apache/skywalking-oap-server:8.8.1
EOF

timeout $TIMEOUT bash -c -- "\
    while ! kubectl create -f $MANIFEST 2> /dev/null; \
    do \
      sleep 0.1; \
    done"

# make sure the dummy OAPServer will be deleted
trap "kubectl delete OAPServer dummy; rm $MANIFEST" 0 2 3 15   
