# vm-event-based-dns-management
Automatically manage DNS records based on your GCE VM lifecycle events

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

3. gcloud command to deploy your Cloud Function

    gcloud functions deploy {Name} \
        --trigger-topic={PubSub topic}\
        --retry \
        --region={Region} \
        --runtime=go113 \
        --entry-point=PubSubMsgReader \
        --env-vars-file env.yaml

    Example:  
    ```
    gcloud functions deploy gceDnsManager \
        --trigger-topic=gce-vm-events-topic \
        --retry \
        --region=us-central1 \
        --runtime=go113 \
        --entry-point=PubSubMsgReader \
        --env-vars-file env.yaml
    ```