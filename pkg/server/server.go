/*
Copyright 2021 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"knative.dev/pkg/logging"
)

var (
	storagePath = "/tmp"
)

var (
	s = Server{}
)

// Server is used to implement proto.UnimplementedChainsServiceServer
type Server struct {
	logger *zap.SugaredLogger
	proto.UnimplementedChainsServiceServer
	cancel context.CancelFunc
}

func (s *Server) AddEntry(ctx context.Context, req *proto.Entry) (*proto.Empty, error) {
	return &proto.Empty{}, addEntry(req, s.logger)
}

func (s *Server) GetEntryRequest(ctx context.Context, req *proto.EntryRequest) (*proto.Entry, error) {
	return getEntry(req.Uid, s.logger)
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// copied from https://grpc.io/blog/coreos/
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

func Reconcile(ctx context.Context, cfg config.Config) error {
	logger := logging.FromContext(ctx)
	if cfg.Service.Enabled {
		logger.Info("Starting server...")
		if err := s.startServer(ctx, cfg, logger); err != nil {
			return err
		}
		return nil
	}
	return s.stopServer()
}

func (s *Server) startServer(ctx context.Context, cfg config.Config, logger *zap.SugaredLogger) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	gwmux := runtime.NewServeMux()

	if err := proto.RegisterChainsServiceHandlerServer(ctx, gwmux, &Server{logger: logger}); err != nil {
		return errors.Wrap(err, "registering chains service")
	}
	addr := fmt.Sprintf("localhost:%d", cfg.Service.Port)
	dopts := []grpc.DialOption{grpc.WithInsecure()}

	if err := proto.RegisterChainsServiceHandlerFromEndpoint(ctx, gwmux, addr, dopts); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/", gwmux)
	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Service.Port))
	if err != nil {
		return err
	}
	defer conn.Close()

	grpcServer := grpc.NewServer()

	srv := &http.Server{
		Addr:    addr,
		Handler: grpcHandlerFunc(grpcServer, mux),
	}

	logger.Infof("Serving on port %d...", cfg.Service.Port)
	if err := srv.Serve(conn); err != nil {
		return err
	}
	return nil
}

func (s *Server) stopServer() error {
	if s.cancel != nil {
		s.logger.Info("Stopping server...")
		s.cancel()
	}
	return nil
}
