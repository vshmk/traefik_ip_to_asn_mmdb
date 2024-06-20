package traefik_ip_to_asn_mmdb

import (
	"context"
	_ "fmt"
	_ "net/http"
	_ "net/http/httptest"
	"testing"
)

func TestPlugConfig(t *testing.T) {
	plugCfg := CreateConfig()

	plugCfg.mm_asn_db = "./non-existing"
	ResetLookup()
	_, err := New(context.TODO(), nil, plugCfg, "")
	if err != nil {
		t.Fatalf("Must not fail on missing DB")
	}

	plugCfg.mm_asn_db = "README.md"
	_, err = New(context.TODO(), nil, plugCfg, "")
	if err != nil {
		t.Fatalf("Must not fail on invalid DB format")
	}
}
