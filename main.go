package gcedns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
// https://cloud.google.com/functions/docs/calling/pubsub
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// Below struct represents gce_instance auditlog schema
type logMetadata struct {
	InsertID  string `json:"insertId"`
	LogName   string `json:"logName"`
	Operation struct {
		First    bool   `json:"first"`
		ID       string `json:"id"`
		Producer string `json:"producer"`
	} `json:"operation"`
	ProtoPayload struct {
		Type               string `json:"@type"`
		AuthenticationInfo struct {
			PrincipalEmail string `json:"principalEmail"`
		} `json:"authenticationInfo"`
		AuthorizationInfo []struct {
			Granted            bool   `json:"granted"`
			Permission         string `json:"permission"`
			ResourceAttributes struct {
				Name    string `json:"name"`
				Service string `json:"service"`
				Type    string `json:"type"`
			} `json:"resourceAttributes"`
		} `json:"authorizationInfo"`
		MethodName string `json:"methodName"`
		Request    struct {
			Type                       string `json:"@type"`
			CanIPForward               bool   `json:"canIpForward"`
			ConfidentialInstanceConfig struct {
				EnableConfidentialCompute bool `json:"enableConfidentialCompute"`
			} `json:"confidentialInstanceConfig"`
			DeletionProtection bool   `json:"deletionProtection"`
			Description        string `json:"description"`
			Disks              []struct {
				AutoDelete       bool   `json:"autoDelete"`
				Boot             bool   `json:"boot"`
				DeviceName       string `json:"deviceName"`
				InitializeParams struct {
					DiskSizeGb  string `json:"diskSizeGb"`
					DiskType    string `json:"diskType"`
					SourceImage string `json:"sourceImage"`
				} `json:"initializeParams"`
				Mode string `json:"mode"`
				Type string `json:"type"`
			} `json:"disks"`
			DisplayDevice struct {
				EnableDisplay bool `json:"enableDisplay"`
			} `json:"displayDevice"`
			Labels []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"labels"`
			MachineType       string `json:"machineType"`
			Name              string `json:"name"`
			NetworkInterfaces []struct {
				Subnetwork string `json:"subnetwork"`
			} `json:"networkInterfaces"`
			ReservationAffinity struct {
				ConsumeReservationType string `json:"consumeReservationType"`
			} `json:"reservationAffinity"`
			Scheduling struct {
				AutomaticRestart  bool   `json:"automaticRestart"`
				OnHostMaintenance string `json:"onHostMaintenance"`
				Preemptible       bool   `json:"preemptible"`
			} `json:"scheduling"`
			ServiceAccounts []struct {
				Email  string   `json:"email"`
				Scopes []string `json:"scopes"`
			} `json:"serviceAccounts"`
			ShieldedInstanceConfig struct {
				EnableIntegrityMonitoring bool `json:"enableIntegrityMonitoring"`
				EnableSecureBoot          bool `json:"enableSecureBoot"`
				EnableVtpm                bool `json:"enableVtpm"`
			} `json:"shieldedInstanceConfig"`
		} `json:"request"`
		RequestMetadata struct {
			CallerIP                string `json:"callerIp"`
			CallerSuppliedUserAgent string `json:"callerSuppliedUserAgent"`
			DestinationAttributes   struct {
			} `json:"destinationAttributes"`
			RequestAttributes struct {
				Auth struct {
				} `json:"auth"`
				Reason string    `json:"reason"`
				Time   time.Time `json:"time"`
			} `json:"requestAttributes"`
		} `json:"requestMetadata"`
		ResourceLocation struct {
			CurrentLocations []string `json:"currentLocations"`
		} `json:"resourceLocation"`
		ResourceName string `json:"resourceName"`
		Response     struct {
			Type           string `json:"@type"`
			ID             string `json:"id"`
			InsertTime     string `json:"insertTime"`
			Name           string `json:"name"`
			OperationType  string `json:"operationType"`
			Progress       string `json:"progress"`
			SelfLink       string `json:"selfLink"`
			SelfLinkWithID string `json:"selfLinkWithId"`
			StartTime      string `json:"startTime"`
			Status         string `json:"status"`
			TargetID       string `json:"targetId"`
			TargetLink     string `json:"targetLink"`
			User           string `json:"user"`
			Zone           string `json:"zone"`
		} `json:"response"`
		ServiceName string `json:"serviceName"`
	} `json:"protoPayload"`
	ReceiveTimestamp time.Time `json:"receiveTimestamp"`
	Resource         struct {
		Labels struct {
			InstanceID string `json:"instance_id"`
			ProjectID  string `json:"project_id"`
			Zone       string `json:"zone"`
		} `json:"labels"`
		Type string `json:"type"`
	} `json:"resource"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	debug = os.Getenv("DNS_DEBUG")
	// Defualt mode creates DNS records based on VM name. dns_host_name VM label should be unset as well.
	default_mode = false
)

func checkErr(msg string, err error) {
	if err != nil {
		log.Fatalf(msg, ": ", err)
	}
}

// func msgReader() {
// 	ctx := context.Background()

// 	// create a client
// 	client, err := pubsub.NewClient(ctx, "vponnam-ground0")
// 	checkErr("failed to create client", err)

// 	defer client.Close()

// 	var mu sync.Mutex
// 	received := 0

// 	sub := client.Subscription("gce-vm-create-events-topic-sub")

// 	cctx, _ := context.WithCancel(ctx)
// 	err = sub.Receive(cctx,
// 		func(c context.Context, m *pubsub.Message) {
// 			mu.Lock()
// 			defer mu.Unlock()

// 			/* extract details from *pubsub.Message data structure
// 			parse logMessage for required details */
// 			fmt.Printf("%v\n", checkOperation(m.Data, c))

// 			m.Ack()
// 			received++
// 			fmt.Printf("received count: %d\n", received)
// 		})
// 	if err != nil {
// 		checkErr("Error recieving", err)
// 	}
// }

func PubSubMsgReader(ctx context.Context, m PubSubMessage) error {
	logMessage := logMetadata{}
	json.Unmarshal(m.Data, &logMessage)

	result, err := gceEventCheckOperation(m.Data, ctx)
	fmt.Println(result)

	return err
}

// contains all VM/DNS info required to create the DNS record
type DnsInfo struct {
	DnsHostName        string
	DnsZoneName        string
	DnsZoneHostProject string
	DnsDomain          string
	Action             string
	IPs                []string
	VMName             string
	VMProject          string
	PTRZoneHostProject string
	PTRZoneName        string
}

// GCE VM create/delete event processing
func gceEventCheckOperation(data []byte, ctx context.Context) (result string, err error) {
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	if len(data) == 0 {
		return "gceEventCheckOperation received no data", errors.New("error parsing pubsub message")
	}

	logMessage := logMetadata{}
	json.Unmarshal(data, &logMessage)

	if debug != "" {
		fmt.Printf("gceEventCheckOperation received data: %v\n", string(data))
	}

	// Variables used in downstream code
	vm_info, receivedVMData := getGCEMetadata(data, ctx)

	if !receivedVMData {
		log.Println("No VM info received. " + logMessage.ProtoPayload.ResourceName)
		return "No VM info received.", fmt.Errorf("no vm info received: %v", logMessage.ProtoPayload.ResourceName)
	}

	labels := vm_info.Labels
	ips := vm_info.IPs
	vm_name := vm_info.Name

	if debug != "" {
		fmt.Printf("VM_Labels: %v,\t VM_Name: %q,\t  VM_IPs: %v\n", labels, vm_name, ips)
		fmt.Printf("GCE vm info: %v\n", vm_info)
	}

	// validate GCE response
	if len(ips) == 0 {
		fmt.Printf("received no VM IPs for %q\n", logMessage.ProtoPayload.ResourceName)
		return "received no VM IPs", fmt.Errorf("received no VM IPs: %v", logMessage.ProtoPayload.ResourceName)
	}

	for _, task := range logMessage.ProtoPayload.AuthorizationInfo {
		if task.Granted && task.Permission == "compute.instances.create" || logMessage.ProtoPayload.Request.Type == "type.googleapis.com/compute.instanceGroups.addInstances" {
			if labels["dns_skip_record"] == "" {
				dnsCreateInfo := DnsInfo{
					DnsHostName:        labels["dns_host_name"],
					DnsZoneName:        labels["dns_zone_name"],
					DnsZoneHostProject: labels["dns_zone_host_project"],
					DnsDomain:          labels["dns_domain"],
					Action:             "create",
					IPs:                ips,
					VMName:             vm_name,
					VMProject:          vm_info.VMProject,
				}
				var dns_record string

				if dnsManagement(dnsCreateInfo) {
					if dnsCreateInfo.DnsHostName == "" {
						dns_record = dnsCreateInfo.VMName
					} else {
						dns_record = dnsCreateInfo.DnsHostName
					}
					result = fmt.Sprintf("%v's DNS record: %v is created with IP: %v\n", logMessage.ProtoPayload.ResourceName, dns_record, ips)
				} else {
					result = fmt.Sprintf("%v's DNS record: %v is not created\n", logMessage.ProtoPayload.ResourceName, dns_record)
				}
			} else if default_mode { // Default mode ignores VM labels and forces to use the default Zone/Domain values set by the DNS/Admin team.
				dnsCreateInfo := DnsInfo{
					DnsHostName:        vm_name,
					DnsZoneName:        defaultDnsZone,
					DnsZoneHostProject: defaultDnsHostProject,
					DnsDomain:          defaultDnsDomain,
					Action:             "create",
					IPs:                ips,
					VMName:             vm_name,
					VMProject:          vm_info.VMProject,
				}
				// Default mode creates DNS records based on VM names
				if dnsManagement(dnsCreateInfo) {
					result = fmt.Sprintf("%v's DNS record: %v is created with IP: %v\n", logMessage.ProtoPayload.ResourceName, dnsCreateInfo.VMName, ips)
				} else {
					result = fmt.Sprintf("%v's DNS record: %v is not created\n", logMessage.ProtoPayload.ResourceName, dnsCreateInfo.VMName)
				}
			} else {
				result = fmt.Sprintf("dns_skip_record is set for %v\n", logMessage.ProtoPayload.ResourceName)
			}
		} else if task.Granted && task.Permission == "compute.instances.delete" || logMessage.ProtoPayload.Request.Type == "type.googleapis.com/compute.instanceGroups.removeInstances" {
			if labels["dns_skip_record"] == "" {
				dnsDeleteInfo := DnsInfo{
					DnsHostName:        labels["dns_host_name"],
					DnsZoneName:        labels["dns_zone_name"],
					DnsZoneHostProject: labels["dns_zone_host_project"],
					DnsDomain:          labels["dns_domain"],
					Action:             "delete",
					IPs:                ips,
					VMName:             vm_name,
					VMProject:          vm_info.VMProject,
				}
				var dns_record string
				if dnsDeleteInfo.DnsHostName == "" {
					dns_record = dnsDeleteInfo.VMName
				} else {
					dns_record = dnsDeleteInfo.DnsHostName
				}

				if dnsManagement(dnsDeleteInfo) {
					result = fmt.Sprintf("%qs DNS record: %q is deleted for IP: %v\n", logMessage.ProtoPayload.ResourceName, dns_record, ips)
				} else {
					result = fmt.Sprintf("%qs DNS record: %q is not deleted\n", logMessage.ProtoPayload.ResourceName, dns_record)
				}
			}
		}
	}
	return result, nil
}
