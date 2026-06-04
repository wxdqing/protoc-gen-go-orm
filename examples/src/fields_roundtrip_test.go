package src_test

import (
	"testing"

	"github.com/wxdqing/protoc-gen-go-orm/examples/src"
	ormmysql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/mysql"
	ormpgsql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/pgsql"
	"google.golang.org/protobuf/proto"
)

// T-GEN-001：FIELDS 模式简单列 Encode/Decode 往返（不连库）。
// JSON/blob 列在空字节时 DecodeTo 行为见 docs/protoc-gen-go-orm/03-gaps.md，另测。
func TestFieldsMode_Player_EncodeDecodeRoundtrip(t *testing.T) {
	in := &src.Player{
		Id:         42,
		Name:       "roundtrip",
		Level:      9,
		Exp:        100,
		PlayerEnum: src.PlayerEnum_Test1,
		Settings:   map[string]int32{"k": 1},
	}
	row := &ormmysql.Player{}
	if err := row.EncodeFrom(in); err != nil {
		t.Fatalf("EncodeFrom: %v", err)
	}
	out := &src.Player{Id: in.Id}
	if err := row.DecodeTo(out); err != nil {
		t.Fatalf("DecodeTo: %v", err)
	}
	if out.Name != in.Name || out.Level != in.Level || out.Exp != in.Exp {
		t.Fatalf("got name=%q level=%d exp=%d", out.Name, out.Level, out.Exp)
	}
	if out.PlayerEnum != in.PlayerEnum {
		t.Fatalf("enum: got %v", out.PlayerEnum)
	}
	if len(out.Settings) != 1 || out.Settings["k"] != 1 {
		t.Fatalf("settings: %+v", out.Settings)
	}
}

func TestFieldsMode_FieldsPlayer_HerosBlobRoundtrip(t *testing.T) {
	in := &src.FieldsPlayer{
		Id:    7,
		Name:  "blob",
		Heros: []*src.Hero{{Id: 1, Cid: 3, HeroLevel: 5}},
	}
	for _, row := range []interface {
		EncodeFrom(proto.Message) error
		DecodeTo(proto.Message) error
	}{
		&ormmysql.FieldsPlayer{},
		&ormpgsql.FieldsPlayer{},
	} {
		if err := row.EncodeFrom(in); err != nil {
			t.Fatalf("EncodeFrom: %v", err)
		}
		out := &src.FieldsPlayer{Id: in.Id}
		if err := row.DecodeTo(out); err != nil {
			t.Fatalf("DecodeTo: %v", err)
		}
		if len(out.Heros) != 1 || out.Heros[0].Cid != 3 {
			t.Fatalf("heros: %+v", out.Heros)
		}
	}
}
