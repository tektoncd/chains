/*
Copyright 2020 The Tekton Authors
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

package grafeas

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	attestationpb "github.com/grafeas/grafeas/proto/v1beta1/attestation_go_proto"
	commonpb "github.com/grafeas/grafeas/proto/v1beta1/common_go_proto"
	pb "github.com/grafeas/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	gstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logtesting "knative.dev/pkg/logging/testing"
)

type args struct {
	tr        *v1beta1.TaskRun
	payload   []byte
	signature string
	opts      config.StorageOpts
}

type testConfig struct {
	name    string
	args    args
	wantErr bool
}

// This function is to test the implementation of the fake server's ListOccurrences function.
// As the filter logic is implemented, we want to make sure it can be trusted before testing store & retrieve.
func TestBackend_ListOccurrences(t *testing.T) {
	// get grafeas client
	ctx := context.Background()
	conn, client, err := setupConnection()
	if err != nil {
		t.Fatal("Failed to create grafeas client.")
	}
	defer conn.Close()

	// two sample occurrences used for testing
	occs := getExpectedOccurrences()

	// store occurrences into the fake server
	for _, occ := range occs {
		client.CreateOccurrence(ctx, &pb.CreateOccurrenceRequest{Occurrence: occ})
	}

	// construct expected ListOccurrencesResponse
	wantedResponse := &pb.ListOccurrencesResponse{Occurrences: occs}

	// 1. test empty filter string - return all occurrences
	got, err := client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{})
	if err != nil {
		t.Fatal("Failed to call ListOccurrences. error ", err)
	}
	if !cmp.Equal(got, wantedResponse, protocmp.Transform()) {
		t.Errorf("Wrong list of occurrences received for empty filter, got=%s", cmp.Diff(got, wantedResponse, protocmp.Transform()))
	}

	// 2. test multiple chained filter - return multiple occurrences
	got, err = client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{
		Filter: `resourceUrl="/apis/tekton.dev/v1beta1/namespaces/foo1/TaskRun/bar1@uid1" OR resourceUrl="gcr.io/test/kaniko-chains1@sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00"`,
	})
	if err != nil {
		t.Fatal("Failed to call ListOccurrences. error ", err)
	}

	// wanted response should be same as #1 as there are only two occurrences stored
	if !cmp.Equal(got, wantedResponse, protocmp.Transform()) {
		t.Errorf("Wrong list of occurrences received for multiple filters, got=%s", cmp.Diff(got, wantedResponse, protocmp.Transform()))
	}

	// 3. test a single filter - return one occurrence
	got, err = client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{
		Filter: `resourceUrl="gcr.io/test/kaniko-chains1@sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00"`,
	})
	if err != nil {
		t.Fatal("Failed to call ListOccurrences. error ", err)
	}

	wantedResponse = &pb.ListOccurrencesResponse{Occurrences: occs[1:]}
	if !cmp.Equal(got, wantedResponse, protocmp.Transform()) {
		t.Errorf("Wrong list of occurrences received for a single filter, got=%s", cmp.Diff(got, wantedResponse, protocmp.Transform()))
	}
}

/* This function is to test
- if the StorePayload function can create correct occurrences and store them into grafeas server
- if the RetrievePayloads and RetrieveSignatures functions work properly to fetch correct payloads and signatures
*/
func TestBackend_StorePayload(t *testing.T) {
	tests := []testConfig{
		{
			name: "intoto for taskrun, no error",
			args: args{
				tr: &v1beta1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo1",
						Name:      "bar1",
						UID:       types.UID("uid1"),
					},
				},
				payload:   []byte("taskrun payload"),
				signature: "taskrun signature",
				opts:      config.StorageOpts{Key: "taskrun.uuid", PayloadFormat: formats.PayloadTypeInTotoIte6},
			},
			wantErr: false,
		},
		{
			name: "simplesigning for oci, no error",
			args: args{
				tr: &v1beta1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo2",
						Name:      "bar2",
						UID:       types.UID("uid2"),
					},
					Status: v1beta1.TaskRunStatus{
						TaskRunStatusFields: v1beta1.TaskRunStatusFields{
							TaskRunResults: []v1beta1.TaskRunResult{
								// the image digest for test purpose also needs to follow image digest protocol to pass the check.
								// i.e. only contain chars in "sh:0123456789abcdef", and the lenth is 7+64. See more details here:
								// https://github.com/google/go-containerregistry/blob/d9bfbcb99e526b2a9417160e209b816e1b1fb6bd/pkg/name/digest.go#L63
								{Name: "IMAGE_DIGEST", Value: "sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00"},
								{Name: "IMAGE_URL", Value: "gcr.io/test/kaniko-chains1"},
							},
						},
					},
				},
				payload:   []byte("oci payload"),
				signature: "oci signature",
				// The Key field must be the same as the first 12 chars of the image digest.
				// Reason:
				// Inside chains.SignTaskRun function, we set the key field for both artifacts.
				// For OCI artifact, it is implemented as the first 12 chars of the image digest.
				// https://github.com/tektoncd/chains/blob/v0.8.0/pkg/artifacts/signable.go#L200
				opts: config.StorageOpts{Key: "cfe4f0bf41c8", PayloadFormat: formats.PayloadTypeSimpleSigning},
			},
			wantErr: false,
		},
		{
			name: "tekton format for taskrun, error",
			args: args{
				tr: &v1beta1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo3",
						Name:      "bar3",
						UID:       types.UID("uid3"),
					},
				},
				opts: config.StorageOpts{Key: "taskrun2.uuid", PayloadFormat: formats.PayloadTypeTekton},
			},
			wantErr: true,
		},
	}

	ctx := context.Background()

	conn, client, err := setupConnection()
	if err != nil {
		t.Fatal("Failed to create grafeas client.")
	}

	defer conn.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend := Backend{
				logger: logtesting.TestLogger(t),
				tr:     test.args.tr,
				client: client,
				cfg: config.Config{
					Storage: config.StorageConfigs{
						Grafeas: config.GrafeasConfig{
							ProjectID: "test-project",
							NoteID:    "test-note",
						},
					},
				},
			}
			// test if the attestation of the taskrun/oci artifact can be successfully stored into grafeas server
			// and test if payloads and signatures inside the attestation can be retrieved.
			testInterface(ctx, t, test, backend)
		})
	}

	// test if occurrences are created correctly from the StorePayload function
	// - ProjectID field in ListOccurrencesRequest doesn't matter here because we assume there is only one project in the mocked server.
	// - ListOccurrencesRequest with empty filter should be able to fetch all occurrences.
	got, err := client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{})
	if err != nil {
		t.Fatal("Failed to call ListOccurrences. error ", err)
	}
	wantedResponse := &pb.ListOccurrencesResponse{Occurrences: getExpectedOccurrences()}
	if !cmp.Equal(got, wantedResponse, protocmp.Transform()) {
		t.Errorf("Wrong list of occurrences received for empty filter, got=%s", cmp.Diff(got, wantedResponse, protocmp.Transform()))
	}
}

// test attestation storage and retrieval
func testInterface(ctx context.Context, t *testing.T, test testConfig, backend Backend) {
	if err := backend.StorePayload(ctx, test.args.payload, test.args.signature, test.args.opts); (err != nil) != test.wantErr {
		t.Fatalf("Backend.StorePayload() failed. error:%v, wantErr:%v", err, test.wantErr)
	}

	objectIdentifier, err := backend.getResourceURI(test.args.opts)
	if (err != nil) != test.wantErr {
		t.Fatalf("Backend.getResourceURI() failed. error:%v, wantErr:%v", err, test.wantErr)
	}

	// check signature
	expect_signature := map[string][]string{objectIdentifier: []string{test.args.signature}}
	got_signature, err := backend.RetrieveSignatures(ctx, test.args.opts)
	if err != nil {
		t.Fatal("Backend.RetrieveSignatures() failed. error:", err)
	}

	if !cmp.Equal(got_signature, expect_signature) && !test.wantErr {
		t.Errorf("Wrong signature object received, got=%v", cmp.Diff(got_signature, expect_signature))
	}

	// check payload
	expect_payload := map[string]string{objectIdentifier: string(test.args.payload)}
	got_payload, err := backend.RetrievePayloads(ctx, test.args.opts)
	if err != nil {
		t.Fatalf("RetrievePayloads.RetrievePayloads() failed. error:%v", err)
	}

	if !cmp.Equal(got_payload, expect_payload) && !test.wantErr {
		t.Errorf("Wrong payload object received, got=%s", cmp.Diff(got_payload, expect_payload))
	}
}

// Two occurrences that will be used for testing ListOccurrences
func getExpectedOccurrences() []*pb.Occurrence {
	return []*pb.Occurrence{
		// occurrence for taskrun
		{
			// Occurrence Name will be automatically generated by grafeas server based on resource uri.
			// In this fake grafeas server, we mock this behaviour by just using resource URI as the auto-generated occurrence name.
			// In the real world, the auto-generated name will be in the format of `projects/<PROJECT_NAME>/occurrences/<AUTO-GENERATED-ID>`.
			// i.e. projects/my_project/occurrences/06d6e0d6-ee2b-4629-b44a-2188ac92eee4
			Name:     "/apis/tekton.dev/v1beta1/namespaces/foo1/TaskRun/bar1@uid1",
			Resource: &pb.Resource{Uri: "/apis/tekton.dev/v1beta1/namespaces/foo1/TaskRun/bar1@uid1"},
			NoteName: "projects/test-project/notes/test-note",
			Details: &pb.Occurrence_Attestation{
				Attestation: &attestationpb.Details{
					Attestation: &attestationpb.Attestation{
						Signature: &attestationpb.Attestation_GenericSignedAttestation{
							GenericSignedAttestation: &attestationpb.GenericSignedAttestation{
								ContentType:       attestationpb.GenericSignedAttestation_CONTENT_TYPE_UNSPECIFIED,
								SerializedPayload: []byte("taskrun payload"),
								Signatures: []*commonpb.Signature{
									{
										Signature: []byte("taskrun signature"),
										// PublicKeyId: we're only using KMS for signing which is the one we currently set its reference in attestation
									},
								},
							},
						},
					},
				},
			},
			Envelope: &commonpb.Envelope{
				Payload:     []byte("taskrun payload"),
				PayloadType: "in-toto attestations containing a slsa.dev/provenance predicate",
				Signatures: []*commonpb.EnvelopeSignature{
					{
						Sig: []byte("taskrun signature"),
						// PublicKeyId: we're only using KMS for signing which is the one we currently support for storing its reference in attestation
					},
				},
			},
		},
		// occurrence for OCI image
		{
			Name:     "gcr.io/test/kaniko-chains1@sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00",
			Resource: &pb.Resource{Uri: "gcr.io/test/kaniko-chains1@sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00"},
			NoteName: "projects/test-project/notes/test-note",
			Details: &pb.Occurrence_Attestation{
				Attestation: &attestationpb.Details{
					Attestation: &attestationpb.Attestation{
						Signature: &attestationpb.Attestation_GenericSignedAttestation{
							GenericSignedAttestation: &attestationpb.GenericSignedAttestation{
								ContentType:       attestationpb.GenericSignedAttestation_SIMPLE_SIGNING_JSON,
								SerializedPayload: []byte("oci payload"),
								Signatures: []*commonpb.Signature{
									{
										Signature: []byte("oci signature"),
										// PublicKeyId: we're only using KMS for signing which is the one we currently support for storing its reference in attestation
									},
								},
							},
						},
					},
				},
			},
			Envelope: &commonpb.Envelope{
				Payload:     []byte("oci payload"),
				PayloadType: "in-toto attestations containing a slsa.dev/provenance predicate",
				Signatures: []*commonpb.EnvelopeSignature{
					{
						Sig: []byte("oci signature"),
						// PublicKeyId: we're only using KMS for signing which is the one we currently support for storing its reference in attestation
					},
				},
			},
		},
	}
}

// set up the connection between grafeas server and client
// and return the client object to the caller
func setupConnection() (*grpc.ClientConn, pb.GrafeasV1Beta1Client, error) {
	serv := grpc.NewServer()
	pb.RegisterGrafeasV1Beta1Server(serv, &mockGrafeasV1Beta1Server{})

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, nil, err
	}

	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}

	client := pb.NewGrafeasV1Beta1Client(conn)
	return conn, client, nil
}

// --------------------- Mocked GrafeasV1Beta1Server interface -----------------
// https://pkg.go.dev/github.com/grafeas/grafeas@v0.2.0/proto/v1beta1/grafeas_go_proto#GrafeasV1Beta1Server
type mockGrafeasV1Beta1Server struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added in the future.
	pb.UnimplementedGrafeasV1Beta1Server

	// Assume there is only one project for storing notes and occurences
	occurences map[string]*pb.Occurrence
	notes      map[string]*pb.Note
}

func (s *mockGrafeasV1Beta1Server) CreateOccurrence(ctx context.Context, req *pb.CreateOccurrenceRequest) (*pb.Occurrence, error) {
	if s.occurences == nil {
		s.occurences = make(map[string]*pb.Occurrence)
	}

	occID := req.GetOccurrence().GetResource().GetUri()
	expectedResponse := req.GetOccurrence()
	expectedResponse.Name = occID // mock auto-generated id

	s.occurences[occID] = expectedResponse
	return expectedResponse, nil
}

func (s *mockGrafeasV1Beta1Server) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.Note, error) {
	noteID := fmt.Sprintf("%s/notes/%s", req.GetParent(), req.GetNoteId())
	expectedResponse := req.GetNote()
	if s.notes == nil {
		s.notes = make(map[string]*pb.Note)
	}

	if _, exists := s.notes[noteID]; exists {
		return nil, gstatus.Error(codes.AlreadyExists, "note ID already exists")
	}
	s.notes[noteID] = expectedResponse
	return expectedResponse, nil
}

func (s *mockGrafeasV1Beta1Server) ListOccurrences(ctx context.Context, req *pb.ListOccurrencesRequest) (*pb.ListOccurrencesResponse, error) {
	occurrences := []*pb.Occurrence{}

	// if filter string is empty, the expected behaviour will be to return all.
	if len(req.GetFilter()) == 0 {
		for _, occ := range s.occurences {
			occurrences = append(occurrences, occ)
		}
		return &pb.ListOccurrencesResponse{Occurrences: occurrences}, nil
	}

	// if the filter string is not empty, do the filtering.
	// mock how uri filter works
	uris := parseURIFilterString(req.GetFilter())

	for _, uri := range uris {
		for id, occ := range s.occurences {
			if uri == id {
				occurrences = append(occurrences, occ)
			}
		}
	}
	return &pb.ListOccurrencesResponse{Occurrences: occurrences}, nil
}

// parse a chained uri filter string to a list of uris
// example:
// 	- input: `resourceUrl="foo" OR resourceUrl="bar"`
// 	- output: ["foo", "bar"]
func parseURIFilterString(filter string) []string {
	results := []string{}

	// a raw filter string will look like `resourceUrl="foo" OR resourceUrl="bar"`
	// 1. break them into separate statements
	statements := strings.Split(filter, " OR ")

	for _, statement := range statements {
		// each statement looks like `resourceUrl="foo"`
		// 2. extract `foo`
		uriWithQuotes := strings.Split(statement, "=")[1]
		uri := uriWithQuotes[1 : len(uriWithQuotes)-1]
		results = append(results, uri)
	}

	return results
}
