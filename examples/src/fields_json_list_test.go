package src_test

import (
	"testing"

	"github.com/wxdqing/protoc-gen-go-orm/examples/src"
	ormmysql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/mysql"
	ormpgsql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/pgsql"
	"google.golang.org/protobuf/proto"
)

// T-GEN-003：repeated 标量 / enum JSON 列 Encode/Decode 往返。
func TestFieldsMode_Player_RepeatedScalarJSONRoundtrip(t *testing.T) {
	in := &src.Player{
		Id:    1,
		Array: []int32{1, 2, 3},
		Enums: []src.PlayerEnum{src.PlayerEnum_Test1, src.PlayerEnum_Test2},
	}
	cases := []struct {
		name string
		row  interface {
			EncodeFrom(proto.Message) error
			DecodeTo(proto.Message) error
		}
	}{
		{"mysql", &ormmysql.Player{}},
		{"pgsql", &ormpgsql.Player{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.row.EncodeFrom(in); err != nil {
				t.Fatalf("EncodeFrom: %v", err)
			}
			out := &src.Player{Id: in.Id}
			if err := tc.row.DecodeTo(out); err != nil {
				t.Fatalf("DecodeTo: %v", err)
			}
			if len(out.Array) != 3 || out.Array[0] != 1 || out.Array[2] != 3 {
				t.Fatalf("array: %+v", out.Array)
			}
			if len(out.Enums) != 2 || out.Enums[0] != src.PlayerEnum_Test1 || out.Enums[1] != src.PlayerEnum_Test2 {
				t.Fatalf("enums: %+v", out.Enums)
			}
		})
	}
}
