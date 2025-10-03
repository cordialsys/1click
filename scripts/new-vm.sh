#!/bin/bash

set -ex

VM_NAME=build-vm-1
PROJECT=cordialsys-builds

# 4cpu, 16gb ram
MACHINE_TYPE=n2-standard-4
SERVICE_ACCOUNT=cordialsys-builds@cordialsys-builds.iam.gserviceaccount.com
DISK_SIZE=100GB
ZONE=us-east4-a
# ensure machine self-destructs after 2 hours in case we forget to delete it
LIFETIME=2h

gcloud compute instances create ${VM_NAME} \
    --project=${PROJECT} \
    --zone=${ZONE} \
    --machine-type=${MACHINE_TYPE} \
    --network-interface=network-tier=PREMIUM,subnet=default \
    --boot-disk-size=${DISK_SIZE} \
    --boot-disk-type=pd-balanced \
    --service-account=${SERVICE_ACCOUNT} \
    --max-run-duration=${LIFETIME} \
    --instance-termination-action=DELETE \
    --scopes=https://www.googleapis.com/auth/cloud-platform \
    --tags=build-vm \
    --image-family=ubuntu-2204-lts \
    --image-project=ubuntu-os-cloud \
    --metadata=enable-oslogin=TRUE

#gcloud compute ssh ${VM_NAME} --project=${PROJECT} --tunnel-through-iap

#gcloud compute instances delete ${VM_NAME} --project=${PROJECT}