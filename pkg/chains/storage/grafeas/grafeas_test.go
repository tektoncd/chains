/*
Copyright 2022 The Tekton Authors
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
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	pb "github.com/grafeas/grafeas/proto/v1/grafeas_go_proto"
	"github.com/tektoncd/chains/pkg/config"
	gstatus "google.golang.org/grpc/status"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

const (
	ProjectID = "test-project"
	NoteID    = "test-note"
	// repo information for clone taskrun
	repoURL   = "https://github.com/test/tekton-test.git"
	commitSHA = "e02ae3576b4e621bd6798ccbfb89358121d896d7"
	// image information for artifact build task
	artifactURL1    = "gcr.io/test/kaniko-chains1"
	artifactDigest1 = "a2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9"
	artifactURL2    = "us-central1-maven.pkg.dev/test/java"
	artifactDigest2 = "b2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9"
)

// Those variables are
// - the clone taskrun and its provenance
// - the artifact build taskrun and its provenance
// - the CI pipelinerun that mocks as the owner of the two taskruns, and its provenance
var (
	// clone taskrun
	// --------------
	cloneTaskRun = &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "git-clone",
			UID:       types.UID("uid-task1"),
		},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				Results: []v1.TaskRunResult{
					{Name: "CHAINS-GIT_COMMIT", Value: *v1.NewStructuredValues(commitSHA)},
					{Name: "CHAINS-GIT_URL", Value: *v1.NewStructuredValues(repoURL)},
				},
			},
		},
	}

	// clone taskrun provenance
	cloneTaskRunProvenance = intoto.ProvenanceStatement{
		Predicate: slsa.ProvenancePredicate{
			Materials: []common.ProvenanceMaterial{
				{
					URI: repoURL,
					Digest: common.DigestSet{
						"sha1": commitSHA,
					},
				},
			},
		},
	}

	artifactIdentifier1 = fmt.Sprintf("%s@sha256:%s", artifactURL1, artifactDigest1)
	artifactIdentifier2 = fmt.Sprintf("%s@sha256:%s", artifactURL2, artifactDigest2)

	// artifact build taskrun
	buildTaskRun = &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "artifact-build",
			UID:       types.UID("uid-task2"),
		},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				Results: []v1.TaskRunResult{
					{Name: "IMAGE_DIGEST", Value: *v1.NewStructuredValues("sha256:" + artifactDigest1)},
					{Name: "IMAGE_URL", Value: *v1.NewStructuredValues(artifactURL1)},
					{Name: "x_ARTIFACT_DIGEST", Value: *v1.NewStructuredValues("sha256:" + artifactDigest2)},
					{Name: "x_ARTIFACT_URI", Value: *v1.NewStructuredValues(artifactURL2)},
				},
			},
		},
	}

	// artifact built taskrun provenance
	buildTaskRunProvenance = intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Subject: []intoto.Subject{
				{
					Name: artifactURL1,
					Digest: common.DigestSet{
						"sha256": artifactDigest1,
					},
				},
				{
					Name: artifactURL2,
					Digest: common.DigestSet{
						"sha256": artifactDigest2,
					},
				},
			},
		},
	}

	// ci pipelinerun
	ciPipeline = &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ci-pipeline",
			UID:       types.UID("uid-pipeline"),
		},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				Results: []v1.PipelineRunResult{
					// the results from task 1 - clone
					{Name: "CHAINS-GIT_COMMIT", Value: *v1.NewStructuredValues(commitSHA)},
					{Name: "CHAINS-GIT_URL", Value: *v1.NewStructuredValues(repoURL)},
					// the results from task 2 - build
					{Name: "IMAGE_DIGEST", Value: *v1.NewStructuredValues("sha256:" + artifactDigest1)},
					{Name: "IMAGE_URL", Value: *v1.NewStructuredValues(artifactURL1)},
					{Name: "x_ARTIFACT_DIGEST", Value: *v1.NewStructuredValues("sha256:" + artifactDigest2)},
					{Name: "x_ARTIFACT_URI", Value: *v1.NewStructuredValues(artifactURL2)},
				},
			},
		},
	}

	// ci pipelinerun provenance
	ciPipelineRunProvenance = intoto.ProvenanceStatement{
		StatementHeader: buildTaskRunProvenance.StatementHeader,
		Predicate: slsa.ProvenancePredicate{
			Materials: cloneTaskRunProvenance.Predicate.Materials,
		},
	}
)

type args struct {
	runObject objects.TektonObject
	payload   []byte
	signature string
	opts      config.StorageOpts
}

type testConfig struct {
	name            string
	args            args
	wantOccurrences []*pb.Occurrence
	wantErr         bool
}

// This function is to test the implementation of the fake server's ListOccurrences function.
// As the filter logic is implemented, we want to make sure it can be trusted before testing store & retrieve.
func TestBackend_ListOccurrences(t *testing.T) {
	var tests = []struct {
		name            string
		filter          string
		wantOccurrences []*pb.Occurrence
	}{
		{
			name:            "empty filter",
			wantOccurrences: []*pb.Occurrence{getPipelineRunBuildOcc(t, artifactIdentifier1), getPipelineRunBuildOcc(t, artifactIdentifier2)},
		},
		{
			name:            "multiple filters",
			filter:          fmt.Sprintf(`resourceUrl="%s" OR resourceUrl="%s"`, artifactIdentifier1, artifactIdentifier2),
			wantOccurrences: []*pb.Occurrence{getPipelineRunBuildOcc(t, artifactIdentifier1), getPipelineRunBuildOcc(t, artifactIdentifier2)},
		},
		{
			name:            "a single filter",
			filter:          fmt.Sprintf(`resourceUrl="%s"`, artifactIdentifier1),
			wantOccurrences: []*pb.Occurrence{getPipelineRunBuildOcc(t, artifactIdentifier1)},
		},
	}

	// setup
	ctx, _ := rtesting.SetupFakeContext(t)

	conn, client, err := setupConnection()
	if err != nil {
		t.Fatal("Failed to create grafeas client.")
	}
	defer conn.Close()

	// store two sample occurrences into the fake server
	occs := []*pb.Occurrence{getPipelineRunBuildOcc(t, artifactIdentifier1), getPipelineRunBuildOcc(t, artifactIdentifier2)}

	for _, occ := range occs {
		// create note
		_, err := client.CreateNote(ctx, &pb.CreateNoteRequest{
			Parent: "projects/" + ProjectID,
			NoteId: fmt.Sprintf("%s-pipelinerun-intoto", NoteID),
			Note: &pb.Note{
				Name: occ.NoteName,
			},
		})
		if err != nil && status.Code(err) != codes.AlreadyExists {
			t.Fatal("Failed to create notes in the server, err:", err)
		}

		// create occurrence
		_, err = client.CreateOccurrence(ctx, &pb.CreateOccurrenceRequest{Occurrence: occ})
		if err != nil {
			t.Fatal("Failed to create occurrence in the server, err:", err)
		}
	}

	for _, tc := range tests {
		got, err := client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{Filter: tc.filter})
		if err != nil {
			t.Fatalf("Failed to call ListOccurrences: %v", err)
		}

		want := &pb.ListOccurrencesResponse{Occurrences: tc.wantOccurrences}
		if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
			t.Errorf("Wrong list of occurrences received, diff=%s", diff)
		}
	}
}

// This function is to test
// - if the StorePayload function can create correct occurrences and store them into grafeas server
// - if the RetrievePayloads and RetrieveSignatures functions work properly to fetch correct payloads and signatures
func TestGrafeasBackend_StoreAndRetrieve(t *testing.T) {
	tests := []testConfig{
		{
			name: "intoto for clone taskrun, no error, no occurrences created because no artifacts were built.",
			args: args{
				runObject: &objects.TaskRunObjectV1{
					TaskRun: cloneTaskRun,
				},
				payload:   getRawPayload(t, cloneTaskRunProvenance),
				signature: "clone taskrun signatures",
				opts:      config.StorageOpts{PayloadFormat: formats.PayloadTypeSlsav1},
			},
			wantOccurrences: nil,
			wantErr:         false,
		},
		{
			name: "intoto for build taskrun, no error, 2 BUILD occurrences should be created for the 2 artifacts generated.",
			args: args{
				runObject: &objects.TaskRunObjectV1{
					TaskRun: buildTaskRun,
				},
				payload:   getRawPayload(t, buildTaskRunProvenance),
				signature: "build taskrun signature",
				opts:      config.StorageOpts{PayloadFormat: formats.PayloadTypeSlsav1},
			},
			wantOccurrences: []*pb.Occurrence{getTaskRunBuildOcc(t, artifactIdentifier1), getTaskRunBuildOcc(t, artifactIdentifier2)},
			wantErr:         false,
		},
		{
			name: "simplesigning for the build taskrun, no error, 1 ATTESTATION occurrence should be created for the artifact specified in storageopts.key",
			args: args{
				runObject: &objects.TaskRunObjectV1{
					TaskRun: buildTaskRun,
				},
				payload:   []byte("attestation payload"),
				signature: "build taskrun image signature",
				opts:      config.StorageOpts{FullKey: artifactIdentifier1, PayloadFormat: formats.PayloadTypeSimpleSigning},
			},
			wantOccurrences: []*pb.Occurrence{getTaskRunAttestationOcc(t, artifactIdentifier1)},
			wantErr:         false,
		},
		{
			name: "intoto for the ci pipeline, no error, 2 occurences should be created for the pipelinerun for the 2 artifact generated.",
			args: args{
				runObject: &objects.PipelineRunObjectV1{
					PipelineRun: ciPipeline,
				},
				payload:   getRawPayload(t, ciPipelineRunProvenance),
				signature: "ci pipelinerun signature",
				opts:      config.StorageOpts{PayloadFormat: formats.PayloadTypeSlsav1},
			},
			wantOccurrences: []*pb.Occurrence{getPipelineRunBuildOcc(t, artifactIdentifier1), getPipelineRunBuildOcc(t, artifactIdentifier2)},
			wantErr:         false,
		},
		{
			name: "tekton format for a taskrun, error, only simplesigning and intoto are supported",
			args: args{
				runObject: &objects.TaskRunObjectV1{
					TaskRun: buildTaskRun,
				},
				payload:   []byte("foo"),
				signature: "bar",
				opts:      config.StorageOpts{PayloadFormat: formats.PayloadTypeTekton},
			},
			wantOccurrences: nil,
			wantErr:         true,
		},
	}

	// setup connection
	ctx, _ := rtesting.SetupFakeContext(t)
	conn, client, err := setupConnection()
	if err != nil {
		t.Fatal("Failed to create grafeas client.")
	}
	defer conn.Close()

	// collect all the occurences expected to be created in the server
	allOccurrencesInServer := []*pb.Occurrence{}
	for _, test := range tests {

		// run the test
		t.Run(test.name, func(t *testing.T) {
			ctx := logging.WithLogger(ctx, logtesting.TestLogger(t))
			backend := Backend{
				client: client,
				cfg: config.Config{
					Storage: config.StorageConfigs{
						Grafeas: config.GrafeasConfig{
							ProjectID: ProjectID,
							NoteID:    NoteID,
						},
					},
					Artifacts: config.ArtifactConfigs{
						PipelineRuns: config.Artifact{
							DeepInspectionEnabled: false,
						},
					},
				},
			}
			// test if the attestation of the taskrun/oci artifact can be successfully stored into grafeas server
			// and test if payloads and signatures inside the attestation can be retrieved.
			testStoreAndRetrieveHelper(ctx, t, test, backend)

			// accumulate the expected occurrences from each call.
			allOccurrencesInServer = append(allOccurrencesInServer, test.wantOccurrences...)
		})
	}

	// test if all occurrences are created correctly from multiple requests
	// - ProjectID field in ListOccurrencesRequest doesn't matter here because we assume there is only one project in the mocked server.
	// - ListOccurrencesRequest with empty filter should be able to fetch all occurrences.
	got, err := client.ListOccurrences(ctx, &pb.ListOccurrencesRequest{})
	if err != nil {
		t.Fatal("Failed to call ListOccurrences. error ", err)
	}

	sort.Slice(allOccurrencesInServer, func(i, j int) bool {
		return allOccurrencesInServer[i].ResourceUri+allOccurrencesInServer[i].NoteName < allOccurrencesInServer[j].ResourceUri+allOccurrencesInServer[j].NoteName
	})
	want := &pb.ListOccurrencesResponse{
		Occurrences: allOccurrencesInServer,
	}

	if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
		t.Errorf("Wrong list of occurrences received for empty filter, diff=%s", diff)
	}
}

// test attestation storage and retrieval
func testStoreAndRetrieveHelper(ctx context.Context, t *testing.T, test testConfig, backend Backend) {
	if err := backend.StorePayload(ctx, test.args.runObject, test.args.payload, test.args.signature, test.args.opts); (err != nil) != test.wantErr {
		t.Fatalf("Backend.StorePayload() failed. error:%v, wantErr:%v", err, test.wantErr)
	}

	// if occurrence is not expected to be created, then stop and no point to retrieve sig & payload
	if len(test.wantOccurrences) == 0 {
		return
	}

	// check signature
	// ----------------
	expectSignature := map[string][]string{}
	if test.args.opts.PayloadFormat == formats.PayloadTypeSimpleSigning {
		expectSignature[test.args.opts.FullKey] = []string{test.args.signature}
	}
	if _, ok := formats.IntotoAttestationSet[test.args.opts.PayloadFormat]; ok {
		allURIs := extract.RetrieveAllArtifactURIs(ctx, test.args.runObject, false)
		for _, u := range allURIs {
			expectSignature[u] = []string{test.args.signature}
		}
	}

	gotSignature, err := backend.RetrieveSignatures(ctx, test.args.runObject, test.args.opts)
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
		expectPayload[test.args.opts.FullKey] = string(test.args.payload)
	}
	if _, ok := formats.IntotoAttestationSet[test.args.opts.PayloadFormat]; ok {
		allURIs := extract.RetrieveAllArtifactURIs(ctx, test.args.runObject, false)
		for _, u := range allURIs {
			expectPayload[u] = string(test.args.payload)
		}
	}

	gotPayload, err := backend.RetrievePayloads(ctx, test.args.runObject, test.args.opts)
	if err != nil {
		t.Fatal("RetrievePayloads.RetrievePayloads() failed: ", err)
	}

	if diff := cmp.Diff(gotPayload, expectPayload); diff != "" && !test.wantErr {
		t.Errorf("Wrong payload received, diff=%s", diff)
	}
}

// ------------------ occurrences for taskruns and pipelineruns --------------
// BUILD Occurrence for the build taskrun that stores the slsa provenance
func getTaskRunBuildOcc(t *testing.T, identifier string) *pb.Occurrence {
	return &pb.Occurrence{
		Name:        identifier,
		ResourceUri: identifier,
		NoteName:    fmt.Sprintf("projects/%s/notes/%s-taskrun-intoto", ProjectID, NoteID),
		Details: &pb.Occurrence_Build{
			Build: &pb.BuildOccurrence{
				IntotoStatement: &pb.InTotoStatement{
					Subject: []*pb.Subject{
						{
							Name:   artifactURL1,
							Digest: map[string]string{"sha256": artifactDigest1},
						},
						{
							Name:   artifactURL2,
							Digest: map[string]string{"sha256": artifactDigest2},
						},
					},
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
			Payload:     getRawPayload(t, buildTaskRunProvenance),
			PayloadType: "application/vnd.in-toto+json",
			Signatures: []*pb.EnvelopeSignature{
				{Sig: []byte("build taskrun signature")},
			},
		},
	}
}

// ATTESTATION Occurrence for the build taskrun that stores the image attestation
func getTaskRunAttestationOcc(t *testing.T, identifier string) *pb.Occurrence {
	return &pb.Occurrence{
		Name:        identifier,
		ResourceUri: identifier,
		NoteName:    fmt.Sprintf("projects/%s/notes/%s-simplesigning", ProjectID, NoteID),
		Details: &pb.Occurrence_Attestation{
			Attestation: &pb.AttestationOccurrence{
				SerializedPayload: []byte("attestation payload"),
				Signatures: []*pb.Signature{
					{Signature: []byte("build taskrun image signature")},
				},
			},
		},
		Envelope: &pb.Envelope{
			Payload:     []byte("attestation payload"),
			PayloadType: "application/vnd.dev.cosign.simplesigning.v1+json",
			Signatures: []*pb.EnvelopeSignature{
				{Sig: []byte("build taskrun image signature")},
			},
		},
	}
}

func getPipelineRunBuildOcc(t *testing.T, identifier string) *pb.Occurrence {
	return &pb.Occurrence{
		Name:        identifier,
		ResourceUri: identifier,
		NoteName:    fmt.Sprintf("projects/%s/notes/%s-pipelinerun-intoto", ProjectID, NoteID),
		Details: &pb.Occurrence_Build{
			Build: &pb.BuildOccurrence{
				IntotoStatement: &pb.InTotoStatement{
					Subject: []*pb.Subject{
						{
							Name:   artifactURL1,
							Digest: map[string]string{"sha256": artifactDigest1},
						},
						{
							Name:   artifactURL2,
							Digest: map[string]string{"sha256": artifactDigest2},
						},
					},
					Predicate: &pb.InTotoStatement_SlsaProvenanceZeroTwo{
						SlsaProvenanceZeroTwo: &pb.SlsaProvenanceZeroTwo{
							Builder: &pb.SlsaProvenanceZeroTwo_SlsaBuilder{},
							Invocation: &pb.SlsaProvenanceZeroTwo_SlsaInvocation{
								ConfigSource: &pb.SlsaProvenanceZeroTwo_SlsaConfigSource{},
							},
							Materials: []*pb.SlsaProvenanceZeroTwo_SlsaMaterial{
								{
									Uri:    repoURL,
									Digest: map[string]string{"sha1": commitSHA},
								},
							},
						},
					}},
			},
		},
		Envelope: &pb.Envelope{
			Payload:     getRawPayload(t, ciPipelineRunProvenance),
			PayloadType: "application/vnd.in-toto+json",
			Signatures: []*pb.EnvelopeSignature{
				{Sig: []byte("ci pipelinerun signature")},
			},
		},
	}
}

func getRawPayload(t *testing.T, in interface{}) []byte {
	rawPayload, err := json.Marshal(in)
	if err != nil {
		t.Errorf("Unable to marshal the provenance: %v", in)
	}
	return rawPayload
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

	go func() {
		err := serv.Serve(lis)
		if err != nil {
			panic(fmt.Sprintf("failed to setup grafeas connection: %s", err))
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	client := pb.NewGrafeasClient(conn)
	return conn, client, nil
}

// --------------------- Mocked Grafeasv1Server interface -----------------
type mockGrafeasServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added in the future.
	pb.UnimplementedGrafeasServer

	// entries mocks the storage of notes and occurrences.
	// Assume there is only one grafeas project
	entries map[string]*noteOccurrences
}

// noteOccurrences mocks that the behaviour of one note linking multiple occurrences
type noteOccurrences struct {
	// mocks a grafeas note instance
	note *pb.Note
	// mocks occurrences under a note
	// - key is artifact uri (ResourceUri) that represents the occurrence.
	// - value is the actual occurrence instance
	occurrences map[string]*pb.Occurrence
}

func (s *mockGrafeasServer) CreateOccurrence(ctx context.Context, req *pb.CreateOccurrenceRequest) (*pb.Occurrence, error) {
	if s.entries == nil {
		s.entries = make(map[string]*noteOccurrences)
	}

	occ := req.GetOccurrence()
	noteName := req.GetOccurrence().NoteName
	resourceUri := req.GetOccurrence().ResourceUri
	occ.Name = resourceUri // mock how the occurrence ID (name) is outputed.

	if note, ok := s.entries[noteName]; ok {
		if _, ok := note.occurrences[resourceUri]; ok {
			return nil, gstatus.Error(codes.AlreadyExists, "Occurrence ID already exists")
		}
		if note.occurrences == nil {
			note.occurrences = make(map[string]*pb.Occurrence)
		}
		note.occurrences[resourceUri] = occ
		return occ, nil
	}

	return nil, gstatus.Error(codes.FailedPrecondition, "The note for that occurrences does not exist.")
}

func (s *mockGrafeasServer) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.Note, error) {
	notePath := fmt.Sprintf("%s/notes/%s", req.GetParent(), req.GetNoteId())
	noteRequested := req.GetNote()

	if s.entries == nil {
		s.entries = make(map[string]*noteOccurrences)
	}

	if _, ok := s.entries[notePath]; ok {
		return nil, gstatus.Error(codes.AlreadyExists, "note ID already exists")
	}

	s.entries[notePath] = &noteOccurrences{
		note: noteRequested,
	}
	return noteRequested, nil
}

func (s *mockGrafeasServer) ListNotes(ctx context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	notes := []*pb.Note{}
	for _, n := range s.entries {
		notes = append(notes, n.note)
	}
	return &pb.ListNotesResponse{Notes: notes}, nil
}

func (s *mockGrafeasServer) ListOccurrences(ctx context.Context, req *pb.ListOccurrencesRequest) (*pb.ListOccurrencesResponse, error) {
	// to make sure the occurrences we get are in order.
	allOccurrencesInServer := []*pb.Occurrence{}
	for _, note := range s.entries {
		for _, occ := range note.occurrences {
			allOccurrencesInServer = append(allOccurrencesInServer, occ)
		}
	}

	filteredOccs := s.getOccurrencesByFilter(req.GetFilter(), allOccurrencesInServer)
	sort.Slice(filteredOccs, func(i, j int) bool {
		return filteredOccs[i].ResourceUri+filteredOccs[i].NoteName < filteredOccs[j].ResourceUri+filteredOccs[j].NoteName
	})

	return &pb.ListOccurrencesResponse{Occurrences: filteredOccs}, nil
}

func (s *mockGrafeasServer) ListNoteOccurrences(ctx context.Context, req *pb.ListNoteOccurrencesRequest) (*pb.ListNoteOccurrencesResponse, error) {
	noteName := req.Name
	if _, ok := s.entries[noteName]; !ok {
		return nil, nil
	}

	allOccurrences := []*pb.Occurrence{}
	for _, o := range s.entries[noteName].occurrences {
		allOccurrences = append(allOccurrences, o)
	}

	filteredOccs := s.getOccurrencesByFilter(req.GetFilter(), allOccurrences)
	sort.Slice(filteredOccs, func(i, j int) bool {
		return filteredOccs[i].ResourceUri+filteredOccs[i].NoteName < filteredOccs[j].ResourceUri+filteredOccs[j].NoteName
	})
	return &pb.ListNoteOccurrencesResponse{Occurrences: filteredOccs}, nil
}

func (s *mockGrafeasServer) getOccurrencesByFilter(filter string, occurrences []*pb.Occurrence) []*pb.Occurrence {
	// if filter string is empty, the expected behaviour will be to return all.
	if len(filter) == 0 {
		return occurrences
	}

	// if the filter string is not empty, do the filtering.
	// mock how uri filter works
	uris := parseURIFilterString(filter)

	// result result
	result := []*pb.Occurrence{}

	for _, occ := range occurrences {
		for _, uri := range uris {
			if uri == occ.GetResourceUri() {
				result = append(result, occ)
			}
		}
	}

	return result
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
