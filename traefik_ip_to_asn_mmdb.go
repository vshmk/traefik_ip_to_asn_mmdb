package traefik_ip_to_asn_mmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/IncSW/geoip2"
)

type LookupGeoip2 func(ip net.IP) (*GeoipResult, error)

// reader in global var lookup
var lookup_func LookupGeoip2

func ResetLookup() {
	lookup_func = nil
}

type Config struct {
	mm_asn_db             string `json:"MM_ASN_DB"`
	mm_client_asn_header  string `json:"MM_CLIENT_ASN_HEADER"`
	true_client_ip_header string `json:"TRUE_CLIENT_IP_HEADER"`
}

type traefik_mmdb_plugin struct {
	next                  http.Handler
	name                  string
	mm_asn_db             string
	mm_client_asn_header  string
	true_client_ip_header string
}

func CreateConfig() *Config {
	return &Config{
		mm_asn_db:             "./GeoLite2-ASN.mmdb", // Lookup default traefik configuration files path
		mm_client_asn_header:  "X-ASN",
		true_client_ip_header: "True-Client-IP",
	}
}

type GeoipResult struct {
	AutonomousSystemNumber       uint32 `json:"AutonomousSystemNumber"`
	AutonomousSystemOrganization string `json:"AutonomousSystemOrganization"`
}

func CreateDBLookup(reader *geoip2.ASNReader) LookupGeoip2 {
	return func(ip net.IP) (*GeoipResult, error) {
		ret, err := reader.Lookup(ip)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		retval := GeoipResult{
			AutonomousSystemNumber:       ret.AutonomousSystemNumber,
			AutonomousSystemOrganization: ret.AutonomousSystemOrganization,
		}
		return &retval, nil
	}
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if _, err := os.Stat(config.mm_asn_db); err != nil {
		log.Printf("{\"message\":\"[geoip2] DB not found\", \"db\":\"%s\", \"name\":\"%s\", \"err\":\"%v\"}",
			config.mm_asn_db, name, err)
		//This returns valid plugin configuration,
		//so that plugin instance will not crash
		return &traefik_mmdb_plugin{
			next:                  next,
			name:                  name,
			mm_asn_db:             config.mm_asn_db,
			mm_client_asn_header:  config.mm_client_asn_header,
			true_client_ip_header: config.true_client_ip_header,
		}, nil
	}

	if lookup_func == nil && strings.Contains(config.mm_asn_db, "ASN") {
		db, err := geoip2.NewASNReaderFromFile(config.mm_asn_db)
		if err != nil {
			log.Printf("{\"message\":\"[geoip2] DB is not initialized\", \"db\":\"%s\", \"name\":\"%s\", \"err\":\"%v\"}",
				config.mm_asn_db, name, err)
		} else {
			lookup_func = CreateDBLookup(db)
		}
	}

	return &traefik_mmdb_plugin{
		next:                  next,
		name:                  name,
		mm_asn_db:             config.mm_asn_db,
		mm_client_asn_header:  config.mm_client_asn_header,
		true_client_ip_header: config.true_client_ip_header,
	}, nil
}

func (a *traefik_mmdb_plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	//Needed in case there is no valid DB available
	if lookup_func == nil {
		req.Header.Set(a.mm_client_asn_header, "")
		a.next.ServeHTTP(rw, req)
		return
	}

	ip := net.ParseIP(req.Header.Get(a.true_client_ip_header))
	res, err := lookup_func(ip)

	if err != nil {
		req.Header.Set(a.mm_client_asn_header, "")
	} else {
		res_json, _ := json.Marshal(res)
		req.Header.Set(a.mm_client_asn_header, string(res_json))
	}

	a.next.ServeHTTP(rw, req)
}
