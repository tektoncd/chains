#!/usr/bin/env bash
# Copyright 2021 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -e

docker build -t gen-proto -f hack/proto/Dockerfile --target generate-files .

# Copy files
docker run --rm gen-proto cat /proto/github.com/tektoncd/chains/proto/chains.pb.go > proto/chains.pb.go
docker run --rm gen-proto cat /proto/github.com/tektoncd/chains/proto/chains.pb.gw.go > proto/chains.pb.gw.go

printf "\nFinished generating proto files, please commit the results.\n"
