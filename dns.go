package gcedns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

/* DNS entry management
https://pkg.go.dev/google.golang.org/api/dns/v1#pkg-functions
https://cloud.google.com/dns/docs/reference/v1/changes/create#request-body
*/

// RecordSet
type rdSet struct {
	Kind   string `json:"kind"`
	Rrsets []struct {
		Kind    string   `json:"kind"`
		Name    string   `json:"name"`
		Rrdatas []string `json:"rrdatas"`
		TTL     int      `json:"ttl"`
		Type    string   `json:"type"`
	} `json:"rrsets"`
}

// DNS management default variables
var (
	defaultDnsHostProject = os.Getenv("defaultDnsHostProject")
	defaultDnsZone        = os.Getenv("defaultDnsZone")
	defaultDnsDomain      = os.Getenv("defaultDnsDomain")
	defaultPTRDomain      = os.Getenv("defaultPTRDomain")
	defaultPTRZone        = os.Getenv("defaultPTRZone")
	defaultPTRHostProject = os.Getenv("defaultPTRHostProject")
)

// func dnsManagement(action string, dns_host_name string, ips []string) (status bool) {
func dnsManagement(dnsInfo DnsInfo) (status bool) {

	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()
	var (
		dns_host_name  string
		dnsZone        string
		dnsDomain      string
		dnsHostProject string
		ptrHostProject string
		ptrZone        string
		ips            []string
		action         string
	)

	if debug != "" {
		fmt.Printf("dnsInfo: %v\n", dnsInfo)
	}

	// Default wildcard PTRDomain. DNS Zone covering *.in-addr.arpa. domain should pre-exist.
	if defaultPTRDomain == "" {
		defaultPTRDomain = "in-addr.arpa."
	}
	// Populate dns request metadata based on provided VM labels
	if dnsInfo.DnsHostName != "" {
		dns_host_name = dnsInfo.DnsHostName
	} else {
		dns_host_name = dnsInfo.VMName
	}
	if dnsInfo.DnsZoneName != "" {
		dnsZone = dnsInfo.DnsZoneName
	} else {
		dnsZone = defaultDnsZone
	}
	if dnsInfo.DnsDomain != "" {
		dnsDomain = dnsInfo.DnsDomain
	} else {
		dnsDomain = defaultDnsDomain
	}
	if dnsInfo.DnsZoneHostProject != "" {
		dnsHostProject = dnsInfo.DnsZoneHostProject
	} else {
		dnsHostProject = defaultDnsHostProject
	}
	if dnsInfo.PTRZoneHostProject != "" {
		ptrHostProject = dnsInfo.PTRZoneHostProject
	} else {
		ptrHostProject = defaultPTRHostProject
	}
	if dnsInfo.PTRZoneName != "" {
		ptrZone = dnsInfo.PTRZoneName
	} else {
		ptrZone = defaultPTRZone
	}

	ips = dnsInfo.IPs
	action = dnsInfo.Action

	if debug != "" {
		fmt.Printf("From dns.go file: %q\t %q\n", dnsInfo.VMName, dns_host_name)
	}

	if dns_host_name == "" {
		fmt.Println("dns_host_name is null, hence noop")
		return
	} else if dnsInfo.VMProject == "" {
		fmt.Println("VMProject is null, hence noop")
		return
	} else {
		ctx := context.Background()

		dnsService, err := dns.NewService(ctx)
		if err != nil {
			checkErr("Error creating DNS service: ", err)
		}

		// Allow list check
		dns_name := fmt.Sprintf(dns_host_name + "." + dnsDomain)
		if !checkAllowList(dns_name, dnsInfo.VMProject) {
			fmt.Printf("%q is not in the allow list for %q\n", dns_name, dnsInfo.VMProject)
			return false
		} else if len(ips) == 0 {
			fmt.Printf("%q returned no IPs: %v\n", dnsInfo.VMName, ips)
		}

		/* change record
		conditional recordSet support can be added - TDB */
		dnsRecordSet := &dns.ResourceRecordSet{
			Name:             dns_host_name + "." + dnsDomain,
			Rrdatas:          ips,
			SignatureRrdatas: []string{},
			Ttl:              60,
			Type:             "A",
		}

		// PTR record is created for VM's eth0 primary IP
		ptrRecordSet := &dns.ResourceRecordSet{
			Name:    ptrRecordConverter(ips[0]),
			Rrdatas: []string{dns_host_name + "." + dnsDomain},
			Ttl:     60,
			Type:    "PTR",
		}

		if action == "create" {
			patch_check := true

			if patch_check {
				// check if hostname exist - TDB.
				// If exists patch an existing record - TBD.
				recordSetName, err := dnsService.ResourceRecordSets.List(dnsHostProject, dnsZone).Name(dns_host_name + "." + dnsDomain).Do()
				if err != nil {
					checkErr("Error listing RecordSets: ", err)
				}
				rs_resp, err := recordSetName.MarshalJSON()
				if err != nil {
					checkErr("Error parsing RecordSet response: ", err)
				}
				// fmt.Printf("RS list resp: %v\n", string(rs_resp))

				rsdata := rdSet{}
				if err = json.Unmarshal(rs_resp, &rsdata); err != nil {
					checkErr("Error unmarshalling recordSet "+dns_name+": ", err)
				}
				// PATCH call to update existing entry - TBD.
				// Support other types of records, ex: CNAME
				for _, record := range rsdata.Rrsets {
					if record.Name == dns_name {
						// Create PTR record
						ptrcreateChange := &dns.Change{
							Additions: []*dns.ResourceRecordSet{ptrRecordSet},
						}
						if !dnsChange(ptrHostProject, ptrZone, ptrcreateChange) {
							fmt.Printf("Error creating PTR record: %q\n", ptrRecordSet.Rrdatas[0])
						}
						// Patch A record
						return patchRS(dnsHostProject, dnsZone, dns_name, "A", ipCreateChecker(record.Rrdatas, ips))
					}
				}
			}

			createChange := &dns.Change{
				Additions: []*dns.ResourceRecordSet{dnsRecordSet},
			}
			if !dnsChange(dnsHostProject, dnsZone, createChange) {
				fmt.Printf("Error creating A record: %q\n", dnsRecordSet.Name)
			}

			ptrcreateChange := &dns.Change{
				Additions: []*dns.ResourceRecordSet{ptrRecordSet},
			}
			if !dnsChange(ptrHostProject, ptrZone, ptrcreateChange) {
				fmt.Printf("Error creating PTR record: %q\n", ptrRecordSet.Rrdatas[0])
			}
		} else if action == "delete" {

			// PATCH check
			patch_check := true

			if patch_check {
				// check if hostname exist - TDB.
				// If exists patch an existing record - TBD.
				recordSetName, err := dnsService.ResourceRecordSets.List(dnsHostProject, dnsZone).Name(dns_host_name + "." + dnsDomain).Do()
				if err != nil {
					checkErr("Error listing RecordSets: ", err)
				}
				rs_resp, err := recordSetName.MarshalJSON()
				if err != nil {
					checkErr("Error parsing RecordSet response: ", err)
				}
				// fmt.Printf("RS list resp: %v\n", string(rs_resp))

				rsdata := rdSet{}
				if err = json.Unmarshal(rs_resp, &rsdata); err != nil {
					checkErr("Error unmarshalling recordSet "+dns_name+": ", err)
				}
				// PATCH call to update existing entry - TBD.
				for _, record := range rsdata.Rrsets {
					if record.Name == dns_name {

						sort.Strings(record.Rrdatas)
						sort.Strings(ips)

						if reflect.DeepEqual(record.Rrdatas, ips) {

							deleteChange := &dns.Change{
								Deletions: []*dns.ResourceRecordSet{dnsRecordSet},
							}
							// _, err = dnsService.Changes.Create(dnsHostProject, dnsZone, deleteChange).Context(ctx).Do()
							// if err != nil {
							// 	checkErr("Error deleting DNS change: ", err)
							// 	return false
							// }
							if !dnsChange(dnsHostProject, dnsZone, deleteChange) {
								fmt.Printf("Error creating A record: %q\n", dnsRecordSet.Name)
							}
							ptrDeleteChange := &dns.Change{
								Deletions: []*dns.ResourceRecordSet{ptrRecordSet},
							}
							if !dnsChange(ptrHostProject, ptrZone, ptrDeleteChange) {
								fmt.Printf("Error deleting PTR record: %q\n", ptrRecordSet.Rrdatas[0])
							}
						} else {
							// Delete PTR record
							ptrDeleteChange := &dns.Change{
								Deletions: []*dns.ResourceRecordSet{ptrRecordSet},
							}
							if !dnsChange(ptrHostProject, ptrZone, ptrDeleteChange) {
								fmt.Printf("Error creating PTR record: %q\n", ptrRecordSet.Rrdatas[0])
							}
							// Patch A record
							return patchRS(dnsHostProject, dnsZone, dns_name, "A", ipDeleteChecker(record.Rrdatas, ips))
						}
					}
				}
			}
		}
	}
	return true
}

/* Helper func to compare IPs for exiting records for create request */
func ipCreateChecker(previous_ips, new_ips []string) (effective_ips []string) {
	// If new IP merge to a single list
	for _, new_ip := range new_ips {
		for _, old_ip := range previous_ips {
			if new_ip == old_ip {
				// IP exists, noop
				return previous_ips
			}
		}
	}
	// New IPs exist, needs a merge
	effective_ips = append(previous_ips, new_ips...)

	return effective_ips
}

/* Helper func to compare IPs for exiting records for delete request */
func ipDeleteChecker(previous_ips, new_ips []string) (effective_ips []string) {

	dedup := make(map[string]bool)

	for _, n_ip := range new_ips {
		dedup[n_ip] = true
	}

	for _, p_ip := range previous_ips {
		if _, exist := dedup[p_ip]; !exist {
			effective_ips = append(effective_ips, p_ip)
		}
	}
	return effective_ips
}

// Helper func to covert IP to PTR record
func ptrRecordConverter(ip string) (ptr_record string) {

	disjoin_ip := strings.Split(ip, ".")

	var (
		ip_ints    []int
		ip_strings []string
	)

	for i := len(disjoin_ip) - 1; i >= 0; i-- {
		int_ip, _ := strconv.Atoi(disjoin_ip[i])
		ip_ints = append(ip_ints, int_ip)
	}

	for _, i := range ip_ints {
		s_ip := strconv.Itoa(i)
		ip_strings = append(ip_strings, s_ip)
	}
	return strings.Join(ip_strings, ".") + "." + defaultPTRDomain
}

type pathBody struct {
	RRdatas []string `json:"rrdatas"`
}

// PATCH implementation for updating records as dns library doesn't cover this
func patchRS(project, zone, dns_name, rs_type string, ips []string) (status bool) {

	if debug != "" {
		fmt.Println("Patch operation details")
		fmt.Printf("Project: %q, Zone: %q, dns_name: %q, rs_type: %q, ips %v\n", project, zone, dns_name, rs_type, ips)
	}

	// oAuth from ADC
	client, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		checkErr("Error creating PATCH client: ", err)
	}

	url := fmt.Sprintf("https://dns.googleapis.com/dns/v1/projects/%v/managedZones/%v/rrsets/%v/%v?alt=json", project, zone, url.QueryEscape(dns_name), rs_type)
	// fmt.Println("PATCH url: " + url)
	body, err := json.Marshal(pathBody{
		RRdatas: ips,
	})
	if err != nil {
		checkErr("Error marshalling PATCH request body: ", err)
	}
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		checkErr("Error patching "+dns_name+": ", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		checkErr("Error patching "+dns_name+": ", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("Non 200 resp code received on patch operation")
		fmt.Printf("PATCH resp: %v\n", string(respBody))
	}
	return resp.StatusCode == 200
}

// DNS record changes
func dnsChange(recordHostProject, recordZone string, dnsChange *dns.Change) (status bool) {

	ctx := context.Background()

	dnsService, err := dns.NewService(ctx)
	if err != nil {
		checkErr("Error creating DNS service: ", err)
	}

	_, err = dnsService.Changes.Create(recordHostProject, recordZone, dnsChange).Context(ctx).Do()
	if err != nil {
		checkErr("Error making DNS change: ", err)
		return false
	}
	return true
}

func checkAllowList(dnsFQDN_Requested, vmProjectID string) bool {
	// Read the allowed dns list from local yaml file
	allowListData, err := ioutil.ReadFile("dns_allow_list.yaml")
	if err != nil {
		log.Fatal(err)
	}
	allow_list := make(map[string]string)

	if err := yaml.Unmarshal(allowListData, &allow_list); err != nil {
		log.Fatal(err)
	}

	if allow_list[vmProjectID] == "" {
		return false
	} else {
		prj_allow, _ := regexp.Compile(allow_list[vmProjectID])
		return prj_allow.MatchString(dnsFQDN_Requested)
	}
}
