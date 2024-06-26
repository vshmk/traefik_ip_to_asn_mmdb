package traefik_ip_to_asn_mmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestDBBasic(t *testing.T) {
	plugCfg := CreateConfig()
	plugCfg.mm_asn_db = "./GeoLite2-ASN.mmdb"

	called := false
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { called = true })

	ResetLookup()
	plugInstance, err := New(context.TODO(), next, plugCfg, "traefik-mmdb-ip-to-asn")
	if err != nil {
		t.Fatalf("Error in creating plugin instance: %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)

	plugInstance.ServeHTTP(recorder, req)
	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("Invalid return code")
	}
	if called != true {
		t.Fatalf("next handler was not called")
	}
}

func TestMissingDB(t *testing.T) {
	plugCfg := CreateConfig()
	plugCfg.mm_asn_db = "./missing"

	called := false
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { called = true })

	ResetLookup()
	plugInstance, err := New(context.TODO(), next, plugCfg, "traefik-mmdb-ip-to-asn")
	if err != nil {
		t.Fatalf("Error in creating plugin instance: %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "1.2.3.4"

	plugInstance.ServeHTTP(recorder, req)
	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("Invalid return code")
	}
	if called != true {
		t.Fatalf("next handler was not called")
	}

	assertHeader(t, req, plugCfg.mm_asn_db, "")
}

func TestDBFromRemoteAddr(t *testing.T) {
	plugCfg := CreateConfig()
	plugCfg.mm_asn_db = "./GeoLite2-ASN.mmdb"

	ValidIP := "188.193.88.199"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	ResetLookup()
	plugInstance, err := New(context.TODO(), next, plugCfg, "traefik-mmdb-ip-to-asn")
	if err != nil {
		t.Fatalf("Error in creating plugin instance: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = fmt.Sprintf("%s:9999", ValidIP)
	req.Header.Set(plugCfg.true_client_ip_header, ValidIP)

	recorder := httptest.NewRecorder()
	plugInstance.ServeHTTP(recorder, req)

	expected, _ := json.Marshal(GeoipResult{AutonomousSystemNumber: 3209, AutonomousSystemOrganization: "Vodafone GmbH"})
	key := plugCfg.mm_client_asn_header
	if req.Header.Get(key) != string(expected) {
		t.Fatalf("invalid value of header [%s] != %s", key, req.Header.Get(key))
	}
}

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()
	if req.Header.Get(key) != expected {
		t.Fatalf("invalid value of header [%s] != %s", key, req.Header.Get(key))
	}
}
