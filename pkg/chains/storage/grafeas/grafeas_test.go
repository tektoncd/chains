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
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/testing/protocmp"

	pb "github.com/grafeas/grafeas/proto/v1/grafeas_go_proto"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	gstatus "google.golang.org/grpc/status"
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
		_, err := client.CreateOccurrence(ctx, &pb.CreateOccurrenceRequest{Occurrence: occ})
		if err != nil {
			t.Fatal("Failed to create occurrence in the server")
		}
	}

	// construct expected ListOccurrencesResponse
	wantedResponse := &pb.ListOccurrencesResponse{Occurrences: occs}

	// 1. test empty filter string - return all occurrences
	got, err := client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{})
	if err != nil {
		t.Fatalf("Failed to call ListOccurrences: %v", err)
	}
	if diff := cmp.Diff(got, wantedResponse, protocmp.Transform()); diff != "" {
		t.Errorf("Wrong list of occurrences received for empty filter, diff=%s", diff)
	}

	// 2. test multiple chained filter - return multiple occurrences
	got, err = client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{
		Filter: `resourceUrl="gcr.io/test/kaniko-chains1@sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00" OR resourceUrl="gcr.io/test/kaniko-chains1@sha256:a2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9"`,
	})
	if err != nil {
		t.Fatalf("Failed to call ListOccurrences: %v", err)
	}

	wantedResponse = &pb.ListOccurrencesResponse{Occurrences: occs[:2]}
	if diff := cmp.Diff(got, wantedResponse, protocmp.Transform()); diff != "" {
		t.Errorf("Wrong list of occurrences received for multiple filters, diff=%s", diff)
	}

	// 3. test a single filter - return one occurrence
	got, err = client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{
		Filter: `resourceUrl="gcr.io/test/kaniko-chains1@sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00"`,
	})
	if err != nil {
		t.Fatalf("Failed to call ListOccurrences: %v", err)
	}

	wantedResponse = &pb.ListOccurrencesResponse{Occurrences: occs[1:2]}
	if diff := cmp.Diff(got, wantedResponse, protocmp.Transform()); diff != "" {
		t.Errorf("Wrong list of occurrences received for a single filter, diff=%s", diff)
	}
}

/*
	This function is to test

- if the StorePayload function can create correct occurrences and store them into grafeas server
- if the RetrievePayloads and RetrieveSignatures functions work properly to fetch correct payloads and signatures
*/
func TestGrafeasBackend_StoreAndRetrieve(t *testing.T) {
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
					Status: v1beta1.TaskRunStatus{
						TaskRunStatusFields: v1beta1.TaskRunStatusFields{
							TaskRunResults: []v1beta1.TaskRunResult{
								// the image digest for test purpose also needs to follow image digest protocol to pass the check.
								// i.e. only contain chars in "sh:0123456789abcdef", and the lenth is 7+64. See more details here:
								// https://github.com/google/go-containerregistry/blob/d9bfbcb99e526b2a9417160e209b816e1b1fb6bd/pkg/name/digest.go#L63
								{Name: "IMAGE_DIGEST", Value: *v1beta1.NewArrayOrString("sha256:a2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9")},
								{Name: "IMAGE_URL", Value: *v1beta1.NewArrayOrString("gcr.io/test/kaniko-chains1")},
								{Name: "x_ARTIFACT_DIGEST", Value: *v1beta1.NewArrayOrString("sha256:b2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9")},
								{Name: "x_ARTIFACT_URI", Value: *v1beta1.NewArrayOrString("us-central1-maven.pkg.dev/test/java")},
							},
						},
					},
				},
				payload:   []byte("{}"),
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
								{Name: "IMAGE_DIGEST", Value: *v1beta1.NewArrayOrString("sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00")},
								{Name: "IMAGE_URL", Value: *v1beta1.NewArrayOrString("gcr.io/test/kaniko-chains1")},
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
			testStoreAndRetrieve(ctx, t, test, backend)
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
	if diff := cmp.Diff(got, wantedResponse, protocmp.Transform()); diff != "" {
		t.Errorf("Wrong list of occurrences received for empty filter, diff=%s", diff)
	}
}

// test attestation storage and retrieval
func testStoreAndRetrieve(ctx context.Context, t *testing.T, test testConfig, backend Backend) {
	if err := backend.StorePayload(ctx, test.args.tr, test.args.payload, test.args.signature, test.args.opts); (err != nil) != test.wantErr {
		t.Fatalf("Backend.StorePayload() failed. error:%v, wantErr:%v", err, test.wantErr)
	}

	// check signature
	// ----------------
	expectSignature := map[string][]string{}
	if test.args.opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		uri := backend.retrieveSingleOCIURI(test.args.tr, test.args.opts)
		expectSignature[uri] = []string{test.args.signature}
	}
	if test.args.opts.PayloadFormat == formats.PayloadTypeInTotoIte6 {
		allURIs := backend.retrieveAllArtifactIdentifiers(test.args.tr)
		for _, u := range allURIs {
			expectSignature[u] = []string{test.args.signature}
		}
	}

	gotSignature, err := backend.RetrieveSignatures(ctx, test.args.tr, test.args.opts)
	if err != nil {
		t.Fatal("Backend.RetrieveSignatures() failed: ", err)
	}

	if diff := cmp.Diff(gotSignature, expectSignature); diff != "" && !test.wantErr {
		t.Errorf("Wrong signature received, diff=%s", diff)
	}

	// check payload
	// --------------
	expectPayload := map[string]string{}
	if test.args.opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		uri := backend.retrieveSingleOCIURI(test.args.tr, test.args.opts)
		expectPayload[uri] = string(test.args.payload)
	}
	if test.args.opts.PayloadFormat == formats.PayloadTypeInTotoIte6 {
		allURIs := backend.retrieveAllArtifactIdentifiers(test.args.tr)
		for _, u := range allURIs {
			expectPayload[u] = string(test.args.payload)
		}
	}

	gotPayload, err := backend.RetrievePayloads(ctx, test.args.tr, test.args.opts)
	if err != nil {
		t.Fatal("RetrievePayloads.RetrievePayloads() failed: ", err)
	}

	if diff := cmp.Diff(gotPayload, expectPayload); diff != "" && !test.wantErr {
		t.Errorf("Wrong payload received, diff=%s", diff)
	}
}

// Two occurrences that will be used for testing ListOccurrences
func getExpectedOccurrences() []*pb.Occurrence {
	return []*pb.Occurrence{
		// build occurrence for taskrun that is associated with image artifact
		{
			// Occurrence ID will be randomly generated by grafeas server.
			// In this fake grafeas server, we mock this behaviour by just using resource URI as the auto-generated occurrence name.
			Name:        "gcr.io/test/kaniko-chains1@sha256:a2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9",
			ResourceUri: "gcr.io/test/kaniko-chains1@sha256:a2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9",
			NoteName:    "projects/test-project/notes/test-note-intoto",
			Details: &pb.Occurrence_Build{
				Build: &pb.BuildOccurrence{
					IntotoStatement: &pb.InTotoStatement{
						Predicate: &pb.InTotoStatement_SlsaProvenanceZeroTwo{
							SlsaProvenanceZeroTwo: &pb.SlsaProvenanceZeroTwo{
								Builder: &pb.SlsaProvenanceZeroTwo_SlsaBuilder{},
								Invocation: &pb.SlsaProvenanceZeroTwo_SlsaInvocation{
									ConfigSource: &pb.SlsaProvenanceZeroTwo_SlsaConfigSource{},
								},
							},
						}},
				},
			},
			Envelope: &pb.Envelope{
				Payload:     []byte("{}"),
				PayloadType: "application/vnd.in-toto+json",
				Signatures: []*pb.EnvelopeSignature{
					{Sig: []byte("taskrun signature")},
				},
			},
		},
		// attestation occurrence for OCI image
		{
			Name:        "gcr.io/test/kaniko-chains1@sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00",
			ResourceUri: "gcr.io/test/kaniko-chains1@sha256:cfe4f0bf41c80609214f9b8ec0408b1afb28b3ced343b944aaa05d47caba3e00",
			NoteName:    "projects/test-project/notes/test-note-simplesigning",
			Details: &pb.Occurrence_Attestation{
				Attestation: &pb.AttestationOccurrence{
					SerializedPayload: []byte("oci payload"),
					Signatures: []*pb.Signature{
						{Signature: []byte("oci signature")},
					},
				},
			},
			Envelope: &pb.Envelope{
				Payload:     []byte("oci payload"),
				PayloadType: "application/vnd.dev.cosign.simplesigning.v1+json",
				Signatures: []*pb.EnvelopeSignature{
					{Sig: []byte("oci signature")},
				},
			},
		},
		// build occurrence for taskrun that is associated with maven artifact
		{
			Name:        "us-central1-maven.pkg.dev/test/java@sha256:b2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9",
			ResourceUri: "us-central1-maven.pkg.dev/test/java@sha256:b2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9",
			NoteName:    "projects/test-project/notes/test-note-intoto",
			Details: &pb.Occurrence_Build{
				Build: &pb.BuildOccurrence{
					IntotoStatement: &pb.InTotoStatement{
						Predicate: &pb.InTotoStatement_SlsaProvenanceZeroTwo{
							SlsaProvenanceZeroTwo: &pb.SlsaProvenanceZeroTwo{
								Builder: &pb.SlsaProvenanceZeroTwo_SlsaBuilder{},
								Invocation: &pb.SlsaProvenanceZeroTwo_SlsaInvocation{
									ConfigSource: &pb.SlsaProvenanceZeroTwo_SlsaConfigSource{},
								},
							},
						}},
				},
			},
			Envelope: &pb.Envelope{
				Payload:     []byte("{}"),
				PayloadType: "application/vnd.in-toto+json",
				Signatures: []*pb.EnvelopeSignature{
					{Sig: []byte("taskrun signature")},
				},
			},
		},
	}
}

// set up the connection between grafeas server and client
// and return the client object to the caller
func setupConnection() (*grpc.ClientConn, pb.GrafeasClient, error) {
	serv := grpc.NewServer()
	pb.RegisterGrafeasServer(serv, &mockGrafeasServer{})

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, nil, err
	}

	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	client := pb.NewGrafeasClient(conn)
	return conn, client, nil
}

// --------------------- Mocked GrafeasV1Beta1Server interface -----------------
type mockGrafeasServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added in the future.
	pb.UnimplementedGrafeasServer

	// Assume there is only one project for storing notes and occurences
	occurences map[string]*pb.Occurrence
	notes      map[string]*pb.Note
}

func (s *mockGrafeasServer) CreateOccurrence(ctx context.Context, req *pb.CreateOccurrenceRequest) (*pb.Occurrence, error) {
	if s.occurences == nil {
		s.occurences = make(map[string]*pb.Occurrence)
	}

	occID := req.GetOccurrence().GetResourceUri()
	expectedResponse := req.GetOccurrence()
	expectedResponse.Name = occID // mock auto-generated id

	s.occurences[occID] = expectedResponse
	return expectedResponse, nil
}

func (s *mockGrafeasServer) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.Note, error) {
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

func (s *mockGrafeasServer) ListOccurrences(ctx context.Context, req *pb.ListOccurrencesRequest) (*pb.ListOccurrencesResponse, error) {
	// to make sure the occurrences we get are in order.
	sortedOccurrencesInServer := []*pb.Occurrence{}
	keys := []string{}
	for k := range s.occurences {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sortedOccurrencesInServer = append(sortedOccurrencesInServer, s.occurences[k])
	}

	// if filter string is empty, the expected behaviour will be to return all.
	if len(req.GetFilter()) == 0 {
		return &pb.ListOccurrencesResponse{Occurrences: sortedOccurrencesInServer}, nil
	}

	// if the filter string is not empty, do the filtering.
	// mock how uri filter works
	uris := parseURIFilterString(req.GetFilter())

	// result occurrences
	occurrences := []*pb.Occurrence{}

	for _, occ := range sortedOccurrencesInServer {
		for _, uri := range uris {
			if uri == occ.GetResourceUri() {
				occurrences = append(occurrences, occ)
			}
		}
	}

	return &pb.ListOccurrencesResponse{Occurrences: occurrences}, nil
}

// parse a chained uri filter string to a list of uris
// example:
//   - input: `resourceUrl="foo" OR resourceUrl="bar"`
//   - output: ["foo", "bar"]
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
