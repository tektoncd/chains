#!/bin/sh
# Copyright 2023 The Tekton Authors
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

export USE_GKE_GCLOUD_AUTH_PLUGIN=True
# The Google Service Account which will be used to access GCP resource like ArtifactRegistry and KMS.
export GSA_NAME=my-gsa
# The Kubernets service account which will be tied to the GSA.
export KSA_NAME=my-ksa
# The name of the repo in GCP's ArtifactRegistry, which will be used to upload artifacts created by this demo.
export CLOUD_REPO=my-repo
# The GCP project_id
export PROJECT_ID=<your-project-here>
# export KO_DOCKER_REPO='gcr.io/${PROJECT_ID}'
# The name of the cluster created in GCP.
export CLUSTER_NAME=slsa-demo-cluster
# The zone in which the cluster is created.
export ZONE=us-central1
# The namespace in which the Tekton resources are created.
export NAMESPACE=default
# The KMS key ring.
export KEYRING=my-keyring
# The KMS key.
export KEY=my-key
# A reference to the KMS key as used in APIs.
export KMS_REF=gcpkms://projects/${PROJECT_ID}/locations/us/keyRings/${KEYRING}/cryptoKeys/${KEY}/cryptoKeyVersions/1
# The builder_id to be used for the build.
export BUILDER_ID=www.example.org/tekton-builder
