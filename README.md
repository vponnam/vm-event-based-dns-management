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
Update `PROJECT_ID` and `ORG_ID` variables in `deploy.sh` and run the below.  
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

