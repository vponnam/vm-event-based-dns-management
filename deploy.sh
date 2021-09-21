#!/bin/bash

set -eo pipefail

# Variables
PROJECT_ID="UPDATEME" #Project where Cloud Function gets deployed
ORG_ID="UPDATEME" #gcloud organizations list --format='value(ID)'

PUBSUB_TOPIC="pb-gce-vm-events-topic"
LOG_SINK_NAME="sk-gce-vm-events"
FUNCTION_NAME="gceDnsManager"
DNS_SA_NAME="dns-admin"
SA_FQDN="${DNS_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# Step1: Enable GCF API
1_enable_apis() {
    gcloud services enable cloudfunctions.googleapis.com \
    --project ${PROJECT_ID} -q
}

# Step2: Create pubsub topic
2_create_pubsub_topic() {
    gcloud pubsub topics create ${PUBSUB_TOPIC} \
    --project ${PROJECT_ID} -q
}

# Step3: Create LogSink
3_create_logsink() {
    gcloud logging sinks create ${LOG_SINK_NAME} pubsub.googleapis.com/projects/${PROJECT_ID}/topics/${PUBSUB_TOPIC} \
    --include-children \
    --organization=${ORG_ID} \
    --log-filter='protoPayload.serviceName="compute.googleapis.com" operation.first="true" protoPayload.methodName=~("compute.instances.insert" OR "compute.instances.delete") protoPayload.request.@type=~("type.googleapis.com/compute.instances.insert" OR "type.googleapis.com/compute.instances.delete")'
}

# Step4: Pubsub IAM permission
4_add_pubsub_iam() {
    gcloud beta pubsub topics add-iam-policy-binding ${PUBSUB_TOPIC} \
    --member=$(gcloud logging sinks describe ${LOG_SINK_NAME} --organization ${ORG_ID} --format='value(writerIdentity)') --role='roles/pubsub.publisher' \
    --project ${PROJECT_ID} -q
}

# Step5: Create SA for DNS management
5_create_sa() {
    gcloud iam service-accounts create ${DNS_SA_NAME} \
    --display-name="DNS SA for GCF" \
    --description="DNS SA for GCF" \
    --project ${PROJECT_ID} -q

    # Assign Project level IAM permissions
    gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:${SA_FQDN}" --role="roles/dns.admin" \
    --project ${PROJECT_ID} -q
}

# Step6: Deploy GCF
6_deploy_cloud_function() {
    gcloud functions deploy ${FUNCTION_NAME} \
        --trigger-topic=${PUBSUB_TOPIC} \
        --retry \
        --region=us-central1 \
        --runtime=go113 \
        --entry-point=PubSubMsgReader \
        --ignore-file=./deploy.sh \
        --env-vars-file env.yaml \
        --service-account=${SA_FQDN} \
        --project ${PROJECT_ID} -q
}

# Deletion steps
#Delete Cloud Functions
7_delete_cloud_function() {
    gcloud functions delete ${FUNCTION_NAME} \
    --project ${PROJECT_ID} -q
}

#Remove pubsub IAM
8_delete_pubsub_iam() {
    gcloud beta pubsub topics remove-iam-policy-binding ${PUBSUB_TOPIC} \
    --member=$(gcloud logging sinks describe ${LOG_SINK_NAME} --organization ${ORG_ID} --format='value(writerIdentity)') --role='roles/pubsub.publisher' \
    --project ${PROJECT_ID}
}

#Delete LogSink
9_delete_log_sink() {
    gcloud logging sinks delete ${LOG_SINK_NAME} \
    --organization ${ORG_ID} -q
}

#Delete PubSub Topic
10_delete_pubsub_topic() {
    gcloud pubsub topics delete ${PUBSUB_TOPIC} \
    --project ${PROJECT_ID} -q
}

#Delete SA created for DNS management
11_delete_sa() {
    #Remove Project level IAM permissions
    gcloud projects remove-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:${SA_FQDN}" --role="roles/dns.admin" \
    --project ${PROJECT_ID} -q

    gcloud iam service-accounts delete ${SA_FQDN} \
    --project ${PROJECT_ID} -q
}

# Deployment steps
case "$1" in 
  deploy)
    echo "enablling API.."; 1_enable_apis
    echo "Creating pubsub topic.."; 2_create_pubsub_topic
    echo "Creating logsink.."; 3_create_logsink
    echo "Adding pubsub IAM.."; 4_add_pubsub_iam
    echo "Creating SA.."; 5_create_sa
    echo "Deploying Cloud Function.."; 6_deploy_cloud_function
  ;;
  delete)
    echo "Deleting Cloud Function"; 7_delete_cloud_function
    echo "Removing PubSub IAM"; 8_delete_pubsub_iam
    echo "Deleting LogSink"; 9_delete_log_sink
    echo "Deleting PubSub Topic "; 10_delete_pubsub_topic
    echo "Deleting SA"; 11_delete_sa
  ;;
  **)
  echo "Supported options: deploy/delete"
  ;;
esac
 


