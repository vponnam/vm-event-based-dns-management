# vm-event-based-dns-management
Programatic management of DNS records for Google Cloud VMs.

## User Running this script should atleast have the below IAM permissions
- Temporary conditional Editor role on the project if possible on the project.

Or below granular IAM roles:  
- Org level [Permission](https://cloud.google.com/logging/docs/export/configure_export_v2#before-you-begin) to create log_sink.
- roles/pubsub.editor
- roles/cloudfunctions.developer
- roles/resourcemanager.projectIamAdmin
- roles/iam.serviceAccountAdmin

## Deploying this code
0. clone this repo to your workspace 
    ``` 
    git clone https://github.com/vponnam/vm-event-based-dns-management.git && cd vm-event-based-dns-management
    ```

1. Update the env.yaml as below with the appropriate values

    Example:
    ```yaml
    defaultDnsHostProject: "prj-c-dnshub-3251"
    defaultDnsZone: "private-domain-zone-name"
    defaultDnsDomain: "gcp.company.com"
    defaultPTRZone: "ptr-zone-name"
    defaultPTRHostProject: "prj-c-dnshub-3251"
    ```

2. Update the dns_allow_list.yaml file with valid details
    
    Example:  
    ```yaml
    prj-dev-4328: "^(devserver|qa).*$"
    ```
    Above regex "^(devserver|qa).*$" will allow hostnames that starts with `devserver*`  or `qa*` for `prj-dev-4328` project. This explicit allow_list functionality is present to allow DNS/Network team to have control(also audit) on the hostnames allowed in a given project, and to avoid situations where arbitary DNS requests are being made across projects that could collide.

3. Deploying Cloud Function  
Export `DNS_PROJECT_ID` and `GCP_ORG_ID` variables locally and run the deploy.sh script.  

    Export variables
    ```sh
    export DNS_PROJECT_ID="PROJECT_ID"
    export GCP_ORG_ID=$(gcloud organizations list --format='value(ID)') 
    ``` 
    or if the user account has access to multiple organizations, explicitly provide the org_id as below   
    ```sh
    export GCP_ORG_ID="ORG_ID"
    ```

    #### Deploy
    ```sh
    ./deploy.sh deploy
    ```
    Below resources are created
    - `cloudfunctions.googleapis.com` API will be enabled
    - PubSub topic. 
    - Org level logSink.
    - PubSub publisher IAM to the logSink Service Account.
    - Create an SA with `roles/dns.admin` for Cloud Function.
    - Deploy the code as a Google Cloud Function. 

    #### Cleanup
    - `cloudfunctions.googleapis.com` API will not be disabled for backward compatibility reasons, if the API is being used by other resources.
    - All remaining resources created above will be deleted.
    ```sh
    ./deploy.sh delete
    ```

## Testing this code in action

### DNS Allow list
Add the valid `project_id` and allowed domains as mentioned in deployment [step2](https://github.com/vponnam/vm-event-based-dns-management#deploying-this-code)

### Deploying through MIG
Launch a MIG with the optional VM labels or with defaults. 

### VM deploy with labels
Example gcloud command to deploy a VM with `dns_host_name` label set: below will create a DNS A record in the given zone/domain with dev01 as the hostname as long as dev01 is allowed for the project as an authorized domain in `dns_allow_list.yaml` file

```sh
gcloud beta compute --project=${ProjectID} instances create ${VMName} \ 
--zone=${zone} \
--machine-type=e2-micro \
--subnet=${Subnet} \
--no-address \
--no-restart-on-failure --maintenance-policy=TERMINATE --preemptible \
--no-service-account --no-scopes --image=debian-10-buster-v20210721 \
--image-project=debian-cloud --boot-disk-size=10GB --boot-disk-type=pd-balanced --boot-disk-device-name=disk-01 \
--shielded-secure-boot --shielded-vtpm \
--shielded-integrity-monitoring \
--labels=dns_host_name=dev01
```

### VM deploy with dns_skip_record label
Example gcloud command to deploy a VM with `dns_skip_record` label set, which will not create any DNS records(including A record).

```sh
gcloud beta compute --project=${ProjectID} instances create ${VMName} \ 
--zone=${zone} --machine-type=e2-micro --subnet=${Subnet} --no-address \
--no-restart-on-failure --maintenance-policy=TERMINATE --preemptible \
--no-service-account --no-scopes --image=debian-10-buster-v20210721 \
--image-project=debian-cloud --boot-disk-size=10GB --boot-disk-type=pd-balanced --boot-disk-device-name=disk-01 \
--shielded-secure-boot --shielded-vtpm \
--shielded-integrity-monitoring \
--labels=dns_skip_record
```