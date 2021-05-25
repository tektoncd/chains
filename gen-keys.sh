#!/usr/bin/env bash

# Copyright 2020 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script calls out to scripts in tektoncd/plumbing to setup a cluster
# and deploy Tekton Pipelines to it for running integration tests.

set -ex

export NAMESPACE=tekton-chains
export SECRET_NAME=signing-secrets

openssl ecparam -genkey -name prime256v1 > ec_private.pem
openssl pkcs8 -topk8 -in ec_private.pem  -nocrypt -out x509.pem
openssl ec -in x509.pem -pubout -out x509.pub

kubectl delete secret ${SECRET_NAME} -n ${NAMESPACE} || true
kubectl create secret generic ${SECRET_NAME} -n ${NAMESPACE} --from-file=x509.pem
