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

FROM golang:1.16@sha256:3c4de86eec9cbc619cdd72424abd88326ffcf5d813a8338a7743c55e5898734f AS generate-files
RUN apt-get update && apt-get install -y unzip moreutils

WORKDIR /protoc
ENV PROTOC_VERSION=3.17.3
RUN wget -O protoc.zip https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
RUN unzip protoc.zip
ENV PATH="/protoc/bin:${PATH}"

WORKDIR /grpc-gateway
RUN wget -q -O- https://github.com/grpc-ecosystem/grpc-gateway/tarball/v1.16.0 | tar --strip-components 1 -zx

WORKDIR /tmp
ENV GOPROXY=https://proxy.golang.org
ENV GO111MODULE=on
RUN go get \
    github.com/golang/protobuf/protoc-gen-go@v1.3.3 \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.16.0 \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v1.16.0 \
    github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@v1.3.2

# Generate proto files for v1 API
WORKDIR /proto
COPY proto/ v2/
RUN protoc \
  -I . \
  -I /protoc/include  \
  -I /grpc-gateway/third_party/googleapis \
  --grpc-gateway_out=logtostderr=true:. \
  --go_out=plugins=grpc:. \
   --doc_out=v2/ \
  --swagger_out=logtostderr=true:. \
  v2/*.proto

# Compare the proto files with the existing proto files
FROM generate-files AS compare
WORKDIR /compare/v1
COPY v1/*.go ./
COPY --from=generate-files /proto/v1/index.md ./
