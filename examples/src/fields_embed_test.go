package src_test

import (
	"testing"

	"github.com/wxdqing/protoc-gen-go-orm/examples/src"
	ormmysql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/mysql"
	ormpgsql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/pgsql"
	"google.golang.org/protobuf/proto"
)

// T-GEN-004：embedded BaseModel / RoleTimestamps 字段级 Encode/Decode 往返。
func TestFieldsMode_EmbeddedBaseModelRoundtrip(t *testing.T) {
	in := &src.Player{
		Id:   9,
		Name: "embed",
		BaseModel: &src.BaseModel{
			CreatedAt: 1_700_000_000_000,
			UpdatedAt: 1_700_000_100_000,
		},
	}
	for name, row := range map[string]interface {
		EncodeFrom(proto.Message) error
		DecodeTo(proto.Message) error
	}{
		"mysql": &ormmysql.Player{},
		"pgsql": &ormpgsql.Player{},
	} {
		t.Run(name, func(t *testing.T) {
			if err := row.EncodeFrom(in); err != nil {
				t.Fatalf("EncodeFrom: %v", err)
			}
			out := &src.Player{Id: in.Id}
			if err := row.DecodeTo(out); err != nil {
				t.Fatalf("DecodeTo: %v", err)
			}
			if out.BaseModel == nil {
				t.Fatal("BaseModel nil")
			}
			if out.BaseModel.CreatedAt != in.BaseModel.CreatedAt || out.BaseModel.UpdatedAt != in.BaseModel.UpdatedAt {
				t.Fatalf("timestamps: got %+v want %+v", out.BaseModel, in.BaseModel)
			}
		})
	}
}

func TestFieldsMode_EmbeddedRoleTimestampsRoundtrip(t *testing.T) {
	in := &src.GameRole{
		ServerId: 1,
		RoleId:   2,
		Timestamps: &src.RoleTimestamps{
			CreatedAt: 1_600_000_000_000,
			UpdatedAt: 1_600_000_500_000,
		},
	}
	cases := []struct {
		name string
		row  interface {
			EncodeFrom(proto.Message) error
			DecodeTo(proto.Message) error
		}
	}{
		{"mysql", &ormmysql.GameRole{}},
		{"pgsql", &ormpgsql.GameRole{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.row.EncodeFrom(in); err != nil {
				t.Fatalf("EncodeFrom: %v", err)
			}
			out := &src.GameRole{ServerId: in.ServerId, RoleId: in.RoleId}
			if err := tc.row.DecodeTo(out); err != nil {
				t.Fatalf("DecodeTo: %v", err)
			}
			if out.Timestamps == nil {
				t.Fatal("Timestamps nil")
			}
			if out.Timestamps.CreatedAt != in.Timestamps.CreatedAt || out.Timestamps.UpdatedAt != in.Timestamps.UpdatedAt {
				t.Fatalf("timestamps: got %+v want %+v", out.Timestamps, in.Timestamps)
			}
		})
	}
}
