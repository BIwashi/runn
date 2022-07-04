package runn

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/grpcstub"
	"github.com/k1LoW/runn/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestGrpcRunner(t *testing.T) {
	tests := []struct {
		name            string
		req             *grpcRequest
		wantReqCount    int
		wantResCount    int
		wantRecvRequest *grpcstub.Request
		wantResMessage  map[string]interface{}
		wantResHeaders  metadata.MD
	}{
		{
			"Unary RPC",
			&grpcRequest{
				service: "grpctest.GrpcTestService",
				method:  "Hello",
				headers: metadata.MD{"3rd": {"stone"}},
				messages: []*grpcMessage{
					{
						op: grpcOpMessage,
						params: map[string]interface{}{
							"name":         "alice",
							"num":          3,
							"request_time": time.Date(2022, 2, 22, 22, 22, 22, 22, time.UTC),
						},
					},
				},
			},
			1,
			1,
			&grpcstub.Request{
				Service: "grpctest.GrpcTestService",
				Method:  "Hello",
				Headers: metadata.MD{
					"content-type": {"application/grpc"},
					"3rd":          {"stone"},
					"user-agent":   {fmt.Sprintf("runn/%s grpc-go/%s", version.Version, grpc.Version)},
				},
				Message: grpcstub.Message{
					"name":         "alice",
					"num":          float64(3),
					"request_time": time.Date(2022, 2, 22, 22, 22, 22, 22, time.UTC).Format(time.RFC3339Nano),
				},
			},
			map[string]interface{}{
				"message":     "hello",
				"num":         float64(32),
				"create_time": time.Date(2022, 6, 25, 5, 24, 43, 861872000, time.UTC).Format(time.RFC3339Nano),
			},
			metadata.MD{
				"content-type": []string{"application/grpc"},
				"hello":        []string{"world"},
			},
		},
		{
			"ServerStreaming RPC",
			&grpcRequest{
				service: "grpctest.GrpcTestService",
				method:  "ListHello",
				headers: metadata.MD{"101000": {"lab"}},
				messages: []*grpcMessage{
					{
						op: grpcOpMessage,
						params: map[string]interface{}{
							"name":         "alice",
							"num":          3,
							"request_time": time.Date(2022, 2, 22, 22, 22, 22, 22, time.UTC),
						},
					},
				},
			},
			1,
			2,
			&grpcstub.Request{
				Service: "grpctest.GrpcTestService",
				Method:  "ListHello",
				Headers: metadata.MD{
					"content-type": {"application/grpc"},
					"101000":       {"lab"},
					"user-agent":   {fmt.Sprintf("runn/%s grpc-go/%s", version.Version, grpc.Version)},
				},
				Message: grpcstub.Message{
					"name":         "alice",
					"num":          float64(3),
					"request_time": time.Date(2022, 2, 22, 22, 22, 22, 22, time.UTC).Format(time.RFC3339Nano),
				},
			},
			map[string]interface{}{
				"message":     "hello",
				"num":         float64(34),
				"create_time": time.Date(2022, 6, 25, 5, 24, 44, 382783000, time.UTC).Format(time.RFC3339Nano),
			},
			metadata.MD{
				"content-type": []string{"application/grpc"},
			},
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := grpcstub.NewServer(t, []string{}, "testdata/grpctest.proto")
			t.Cleanup(func() {
				ts.Close()
			})
			ts.Method("grpctest.GrpcTestService/Hello").
				Header("hello", "world").
				ResponseString(`{"message":"hello", "num":32, "create_time":"2022-06-25T05:24:43.861872Z"}`)
			ts.Method("grpctest.GrpcTestService/ListHello").
				ResponseString(`{"message":"hello", "num":33, "create_time":"2022-06-25T05:24:43.861872Z"}`).
				ResponseString(`{"message":"hello", "num":34, "create_time":"2022-06-25T05:24:44.382783Z"}`)
			o, err := New()
			if err != nil {
				t.Fatal(err)
			}

			r, err := newGrpcRunner("greq", ts.Addr(), o)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Run(ctx, tt.req); err != nil {
				t.Error(err)
			}
			if want := 1; len(r.operator.store.steps) != want {
				t.Errorf("got %v want %v", len(r.operator.store.steps), want)
				return
			}
			{
				got := len(ts.Requests())
				if got != tt.wantReqCount {
					t.Errorf("got %v\nwant %v", got, tt.wantReqCount)
				}
			}
			latest := len(ts.Requests()) - 1
			recvReq := ts.Requests()[latest]
			tt.wantRecvRequest.Headers.Append(":authority", ts.Addr())
			if diff := cmp.Diff(recvReq, tt.wantRecvRequest, nil); diff != "" {
				t.Errorf("%s", diff)
			}

			res := r.operator.store.steps[0]["res"].(map[string]interface{})
			{
				got := len(res["messages"].([]map[string]interface{}))
				if got != tt.wantResCount {
					t.Errorf("got %v\nwant %v", got, tt.wantResCount)
				}
			}
			{
				got := res["message"].(map[string]interface{})
				if diff := cmp.Diff(got, tt.wantResMessage, nil); diff != "" {
					t.Errorf("%s", diff)
				}
			}
			{
				got := res["headers"].(metadata.MD)
				if diff := cmp.Diff(got, tt.wantResHeaders, nil); diff != "" {
					t.Errorf("%s", diff)
				}
			}
		})
	}
}
