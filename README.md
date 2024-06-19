# traefik_ip_to_asn_mmdb
Traefik middleware plugin for ASN lookup in MMDB by request IP
Based on [traefikgoip2](https://github.com/traefik-plugins/traefikgeoip2/tree/main) to get ASN data from [MaxMind GeoIP databases](https://www.maxmind.com/en/geoip2-services-and-databases) and pass it downstream with an HTTP request.

# installation
This plugin relies on [IncSW/geoip2](github.com/IncSW/geoip2)

