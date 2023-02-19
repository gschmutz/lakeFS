package tablediff

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc"
)

func TestDeltaLakeDiffer_Diff(t *testing.T) {
	tests := []struct {
		name         string
		client       TableDifferClientMock
		diffResponse *DiffResponse
		want         Response
		err          error
		expectedErr  error
	}{
		{
			name:   "success",
			client: TableDifferClientMock{},
			diffResponse: &DiffResponse{
				Entries: []*TableOperation{
					{
						Id:        "id1",
						Timestamp: timestamppb.Now(),
						Operation: "OPA",
						Content: map[string]string{
							"this":      "happened",
							"this_also": "happened",
						},
					},
				},
				ChangeType: Type(Changed),
			},
		},
		{
			name:         "failure - table not found",
			client:       TableDifferClientMock{},
			diffResponse: nil,
			err: grpcErr{
				err:  "table is not here",
				code: codes.NotFound,
			},
			expectedErr: ErrTableNotFound,
		},
		{
			name:         "failure - unknown error",
			client:       TableDifferClientMock{},
			diffResponse: nil,
			err: grpcErr{
				err:  "data loss",
				code: codes.DataLoss,
			},
		},
	}
	ctx := context.Background()
	params := Params{
		TablePaths: TablePaths{
			LeftTablePath:  RefPath{Ref: "leftBranch", Path: "table/path"},
			RightTablePath: RefPath{Ref: "rightBranch", Path: "table/path"},
			BaseTablePath:  RefPath{Ref: "baseCommit", Path: "table/path"},
		},
		S3Creds: S3Creds{
			Key:      "niceKey",
			Secret:   "coolSecret",
			Endpoint: "usEast42",
		},
		Repo: "testing-delta-differ-repo",
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.client.loadResponse(tt.diffResponse, tt.err)
			comparedErr := tt.expectedErr
			if comparedErr == nil && tt.err != nil {
				comparedErr = tt.err
			}
			d := &DeltaLakeDiffer{
				client: &tt.client,
			}
			got, err := d.Diff(ctx, params)
			if !errors.Is(err, comparedErr) {
				t.Errorf("Diff() err = %v, expected err = %v", err, comparedErr)
			}
			if comparedErr == nil {
				validateResults(t, got, tt.diffResponse)
			}
		})
	}
}

type testResponse struct {
	dr *DiffResponse
	e  error
}

type TableDifferClientMock struct {
	tr testResponse
}

func (tdc *TableDifferClientMock) TableDiff(ctx context.Context, in *DiffRequest, opts ...grpc.CallOption) (*DiffResponse, error) {
	if tdc.tr.e != nil {
		return nil, tdc.tr.e
	}
	return tdc.tr.dr, nil
}

func (tdc *TableDifferClientMock) ShowHistory(ctx context.Context, in *HistoryRequest, opts ...grpc.CallOption) (*HistoryResponse, error) {
	return nil, nil
}

func (tdc *TableDifferClientMock) loadResponse(dr *DiffResponse, e error) {
	tdc.tr = testResponse{
		dr: dr,
		e:  e,
	}
}

type grpcErr struct {
	err  string
	code codes.Code
}

func (ge grpcErr) Error() string {
	return ge.err
}

func (ge grpcErr) GRPCStatus() *status.Status {
	return status.New(ge.code, ge.err)
}

func validateResults(t *testing.T, resp Response, dr *DiffResponse) {
	if len(resp.Diffs) != len(dr.GetEntries()) {
		t.Errorf("Diff() got = %v, returned from inner op = %v", resp, dr)
	}
	expectedResp := Response{
		ChangeType: ChangeType(dr.ChangeType),
		Diffs:      []DiffEntry{},
	}
	for _, e := range dr.GetEntries() {
		expectedResp.Diffs = append(expectedResp.Diffs, DiffEntry{
			Version:          e.Id,
			Timestamp:        e.Timestamp.AsTime(),
			Operation:        e.Operation,
			OperationContent: e.Content,
		})
	}
	if !reflect.DeepEqual(resp, expectedResp) {
		t.Errorf("Diff() got = %v, want %v", resp, expectedResp)
	}
}