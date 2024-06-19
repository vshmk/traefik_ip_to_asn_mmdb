package traefik_ip_to_asn_mmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/IncSW/geoip2"
)

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

func CreateCofig() *Config {
	return &Config{
		mm_asn_db:             "GeoLite2-ASN.mmdb",
		mm_client_asn_header:  "Default_Results_Header",
		true_client_ip_header: "192.168.0.1",
	}
}

type GeoipResult struct {
	AutonomousSystemNumber       uint32 `json:"AutonomousSystemNumber"`
	AutonomousSystemOrganization string `json:"AutonomousSystemOrganization"`
}

type LookupGeoip2 func(ip net.IP) (*GeoipResult, error)

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

var lookup_func LookupGeoip2

// reader in global var lookup
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if _, err := os.Stat(config.mm_asn_db); err != nil {
		log.Printf("[geoip2] DB not found: db = %s, name = %s, err = %v", config.mm_asn_db, name, err)
		return &traefik_mmdb_plugin{
			next:                  next,
			name:                  name,
			mm_asn_db:             "",
			mm_client_asn_header:  "",
			true_client_ip_header: "",
		}, nil
	}

	if lookup_func == nil {
		db, err := geoip2.NewASNReader([]byte(config.mm_asn_db))
		if err != nil {
			log.Printf("[geoip2] DB is not intialized: db = %s, name = %s, err = %v", config.mm_asn_db, name, err)
		} else {
			lookup_func = CreateDBLookup(db)
		}
	}

	return &traefik_mmdb_plugin{
		next:                  next,
		name:                  name,
		mm_asn_db:             config.mm_asn_db,
		mm_client_asn_header:  config.mm_asn_db,
		true_client_ip_header: config.true_client_ip_header,
	}, nil
}

func (a *traefik_mmdb_plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ip, _, _ := net.ParseCIDR(req.Header.Get(a.true_client_ip_header))
	res, err := lookup_func(ip)

	if err != nil {
		res_json, _ := json.Marshal(res)
		req.Header.Add(a.mm_client_asn_header, string(res_json))
	} else {
		req.Header.Add(a.mm_client_asn_header, "NULL")
	}

	a.next.ServeHTTP(rw, req)
}
