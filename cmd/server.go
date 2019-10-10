// Copyright © 2017 uxbh

package cmd

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/mje-nz/ztdns/dnssrv"
	"github.com/mje-nz/ztdns/ztapi"
)

// serverCmd represents the server command.
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run ztDNS server",
	Long: `Server (ztdns server) will start the DNS server.

	Example: ztdns server`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Check config and bail if anything important is missing.
		if viper.GetBool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		log.Debug("debug: ", viper.GetBool("debug"))
		log.Debugf("interface: %s", viper.GetString("interface"))
		log.Debug("port: ", viper.GetInt("port"))
		log.Debugf("domain: %q", viper.GetString("domain"))
		log.Debug("refresh: ", viper.GetInt("refresh"))
		log.Debug("include-offline: ", viper.GetBool("include-offline"))
		log.Debugf("api-key: %q", viper.GetString("api-key"))
		log.Debugf("api-url: %q", viper.GetString("api-url"))
		log.Debugf("network: %q", viper.GetString("network"))
		log.Debugf("networks: %#v", viper.GetStringMapString("networks"))
		log.Debugf("round-robin: %#v", viper.GetStringMapString("round-robin"))

		if viper.GetString("api-key") == "" {
			return fmt.Errorf("no API key provided")
		}

		if viper.GetString("network") == "" && !viper.IsSet("networks") {
			return fmt.Errorf("no networks configured (specify network or networks)")
		}
		if viper.GetString("network") != "" && viper.IsSet("networks") {
			return fmt.Errorf("conflicting network configuration (specify one of network or networks)")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Update the DNSDatabase
		lastUpdate := updateDNS()
		req := make(chan string)
		// Start the DNS server
		go dnssrv.Start(viper.GetString("interface"), viper.GetInt("port"), viper.GetString("domain"), req)

		refresh := viper.GetInt("refresh")
		for {
			// Block until a new request comes in
			n := <-req
			log.Debugf("Got request for %s", n)
			// If the database hasn't been updated in the last "refresh" minutes, update it.
			if time.Since(lastUpdate) > time.Duration(refresh)*time.Minute {
				log.Infof("DNSDatabase is stale. Refreshing.")
				lastUpdate = updateDNS()
			}
		}
	},
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05 -07:00",
	})

	RootCmd.AddCommand(serverCmd)
	serverCmd.Flags().String("interface", "zt0", "network interface to bind to")
	serverCmd.Flags().Int("port", 53, "port to listen on")
	serverCmd.Flags().String("domain", "zt", "domain to serve names under")
	serverCmd.Flags().Int("refresh", 30, "how often to poll the ZeroTier controller in minutes")
	serverCmd.Flags().Bool("include-offline", true, "include offline members")
	serverCmd.Flags().String("api-key", "", "ZeroTier API key")
	serverCmd.Flags().String("api-url", "https://my.zerotier.com/api", "ZeroTier API URL")
	serverCmd.Flags().String("network", "", "ZeroTier Network ID")
	viper.BindPFlags(serverCmd.Flags())
}

// TODO add tests

// memberNameToDNSLabel converts a ZeroTier member name into a valid DNS label.
// See RFC 1035 section 2.3.1 (https://tools.ietf.org/html/rfc1035).
func memberNameToDNSLabel(name string) string {
	// Convert to lower-case so lookup is case-insensitive
	name = strings.ToLower(name)
	// Labels may consist of letters, digits and hyphens
	name = strings.ReplaceAll(name, " ", "-")
	re := regexp.MustCompile("[^a-z0-9-]+")
	name = re.ReplaceAllString(name, "")
	// TODO must start with a letter
	// TODO must end with a letter or digit
	// TODO must be 63 characters or less
	return name
}

// TODO: refactor
func updateDNS() time.Time {
	// Get config info
	apiKey := viper.GetString("api-key")
	apiUrl := viper.GetString("api-url")
	rootDomain := viper.GetString("domain")
	includeOffline := viper.GetBool("include-offline")

	rrDNSPatterns := make(map[string]*regexp.Regexp)
	rrDNSRecords := make(map[string][]dnssrv.Records)

	for re, host := range viper.GetStringMapString("round-robin") {
		rrDNSPatterns[host] = regexp.MustCompile(re)
		log.Debugf("Creating match '%s' for %s host", re, host)
	}

	networks := viper.GetStringMapString("networks")
	if len(networks) == 0 {
		networks = map[string]string{"": viper.GetString("network")}
	}

	// Get all configured networks:
	for domain, networkID := range networks {
		// TODO: Handle configuration with dots
		suffix := "." + rootDomain + "."
		if domain != "" {
			suffix = "." + domain + suffix
		}

		ztnetwork, err := ztapi.GetNetworkInfo(apiKey, apiUrl, networkID)
		if err != nil {
			log.Fatalf("Unable to get network info: %s", err.Error())
		}

		log.Infof("Getting members of network: %s (%s)", ztnetwork.Config.Name, suffix)
		lst, err := ztapi.GetMemberList(apiKey, apiUrl, ztnetwork.ID)
		if err != nil {
			log.Fatalf("Unable to get member list: %s", err.Error())
		}
		log.Debugf("Got %d members", len(*lst))

		for _, n := range *lst {
			if includeOffline || n.Online {
				// Sanitize member name
				name := memberNameToDNSLabel(n.Name)
				fqdn := name + suffix

				// Clear current DNS records
				dnssrv.DNSDatabase[fqdn] = dnssrv.Records{}
				ip6 := []net.IP{}
				ip4 := []net.IP{}
				// Get 6Plane address if network has it enabled
				if ztnetwork.Config.V6AssignMode.Sixplane {
					ip6 = append(ip6, n.Get6Plane())
				}
				// Get RFC4193 address if network has it enabled
				if ztnetwork.Config.V6AssignMode.Rfc4193 {
					ip6 = append(ip6, n.GetRFC4193())
				}

				// Get the rest of the address assigned to the member
				for _, a := range n.Config.IPAssignments {
					ip4 = append(ip4, net.ParseIP(a))
				}

				dnsRecord := dnssrv.Records{
					A:    ip4,
					AAAA: ip6,
				}

				// Add the FQDN to the database
				log.Infof("Updating %-20s IPv4: %-15s IPv6: %s", fqdn, ip4, ip6)
				dnssrv.DNSDatabase[fqdn] = dnsRecord

				// Finding matches for RoundRobin dns
				for host, re := range rrDNSPatterns {
					log.Debugf("Checking matches for %s host", host)
					if match := re.FindStringSubmatch(n.Name); match != nil {
						rrRecord := host + "." + domain + "." + suffix + "."

						log.Infof("Adding ips to RR record %-15s IPv4: %-15s IPv6: %s, from host %s", rrRecord, ip4, ip6, n.Name)
						rrDNSRecords[rrRecord] = append(rrDNSRecords[rrRecord], dnsRecord)
					}
				}
			}
		}

		for rrRecord, dnsRecords := range rrDNSRecords {
			rrRecordIps := dnssrv.Records{}
			for _, ips := range dnsRecords {
				rrRecordIps.A = append(rrRecordIps.A, ips.A...)
				rrRecordIps.AAAA = append(rrRecordIps.AAAA, ips.AAAA...)
			}

			log.Infof("Updating %-15s IPv4: %-15s IPv6: %s", rrRecord, rrRecordIps.A, rrRecordIps.AAAA)
			dnssrv.DNSDatabase[rrRecord] = rrRecordIps
		}
	}

	// Return the current update time
	return time.Now()
}
