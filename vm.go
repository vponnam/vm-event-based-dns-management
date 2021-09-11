package gcedns

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

type VMInfo struct {
	IPs       []string
	Labels    map[string]string
	Name      string
	VMProject string
}

// call GCE for explicitly retriving any VM metadata
// return - vmlabels map[string]string, vmips []string
func getGCEMetadata(data []byte, ctx context.Context) (vm_info VMInfo, status bool) {
	time.Sleep(5 * time.Second) //adding a little sleep as nic assigment sometimes takes longer.
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	if len(data) == 0 {
		log.Println("No data received")
		return VMInfo{}, false
	}

	logMessage := logMetadata{}
	if err := json.Unmarshal(data, &logMessage); err != nil {
		log.Println("Error parsing logSnippet data in getGCEMetadata()")
		return VMInfo{}, false
	}

	client, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		checkErr("failed initializing the comoute client", err)
	}

	compute, err := compute.New(client)
	// compute, err := compute.NewService(ctx)
	if err != nil {
		checkErr("Error instantiating compute client", err)
	}

	if logMessage.Resource.Labels.ProjectID == "" || logMessage.Resource.Labels.Zone == "" || logMessage.Resource.Labels.InstanceID == "" {
		// fmt.Printf("Missing VM info, returning..\n ProjectID: %q, Zone: %q, InstanceID: %q\n", logMessage.Resource.Labels.ProjectID, logMessage.Resource.Labels.Zone, logMessage.Resource.Labels.InstanceID)
		return VMInfo{}, false
	}

	if debug != "" {
		fmt.Printf("VM Info request parameters:\nProjectID: %v, Zone: %v, InstanceID: %v\n", logMessage.Resource.Labels.ProjectID, logMessage.Resource.Labels.Zone, logMessage.Resource.Labels.InstanceID)
	}

	gce := compute.Instances.Get(logMessage.Resource.Labels.ProjectID, logMessage.Resource.Labels.Zone, logMessage.Resource.Labels.InstanceID)

	vm, err := gce.Do()
	if err != nil {
		// In case VM does't exist, this is a noop.
		fmt.Printf("Error occured: %v\n", err)
		return VMInfo{}, false
	}

	var vmips []string
	for _, ips := range vm.NetworkInterfaces {
		vmips = append(vmips, ips.NetworkIP)
	}

	vm_info = VMInfo{
		IPs:       vmips,
		Labels:    vm.Labels,
		Name:      vm.Name,
		VMProject: logMessage.Resource.Labels.ProjectID,
	}

	// vm.Hostname for hostname. return vm.Labels, vmips
	return vm_info, true
}
