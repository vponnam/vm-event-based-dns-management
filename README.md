# vm-event-based-dns-management
Automatically manage DNS records based on your GCE VM lifecycle events

## Deploying this code
1. Create a file named env.yaml as below, and fillout respective values

    ```yaml
    defaultDnsHostProject: 
    defaultDnsZone: 
    defaultDnsDomain: 
    defaultPTRZone: 
    defaultPTRHostProject: 
    ```

    Example:

    ```yaml
    defaultDnsHostProject: "prj-c-dnshub-3251"
    defaultDnsZone: "private-domain-zone-name"
    defaultDnsDomain: "gcp.company.com"
    defaultPTRZone: "ptr-zone-name"
    defaultPTRHostProject: "prj-c-dnshub-3251"
    ```


2. gcloud command to deploy your Cloud Function

    gcloud functions deploy {Name} \
        --trigger-topic={PubSub topic}\
        --retry \
        --region={Region} \
        --runtime=go113 \
        --entry-point=PubSubMsgReader \
        --env-vars-file env.yaml

    Example:  
    ```
    gcloud functions deploy PubSubMsgReader \
        --trigger-topic=gce-vm-create-events-topic \
        --retry \
        --region=us-central1 \
        --runtime=go113 \
        --entry-point=PubSubMsgReader \
        --env-vars-file env.yaml
    ```