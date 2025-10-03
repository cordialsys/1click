# 1Click

This is the VM and "Panel" server to make deploying Treasury very painless. No SSH access or any manually configuration.
A box to set up once and leave it in the "basement".

This handles installing an initial version of Treasury, and pairing it with other node(s) that are being securely hosted elsewhere.
All you need to bring is a VM to run this on, and an activation key from Cordial Systems.

Backup/restore procedures can be fully handled run using the built-in UI to this VM. Encrypt-at-rest can be configured with any compatible
secret manager (GCP, AWS, Vault, etc).

## Running

We maintain marketplace listings for this image on various cloud platforms.

- [AWS](https://aws.amazon.com/marketplace/pp/prodview-xbqazqjdueayu)
- GCP (Coming soon)

### Setup

After launching a VM, you need to configure a port forward so you can access the Panel UI to complete setup.

#### AWS

#### GCP

## Building

The VM is built as a [bootable container](https://docs.fedoraproject.org/en-US/bootc/getting-started/).
The container is then converted to a VM image for the target platform to be booted initially. Future signed updates to
VM's can be applied directly from the container registry.

These instructions are provided only as a _reference_ in case you wish to build the VM yourself.

Build container image for your target platform.

```bash
BASE=aws docker buildx bake
```

Now convert the image to a VM image using Podman and [bootc image builder](https://github.com/osbuild/bootc-image-builder).

### For GCP

```bash
mkdir -p output
podman run \
    --rm \
    -it \
    --privileged \
    --pull=newer \
    --security-opt label=disable \
    -v ./output:/output \
    -v /var/lib/containers/storage:/var/lib/containers/storage \
    quay.io/centos-bootc/bootc-image-builder:latest \
    --type gce \
	--use-librepo=True \
   YOUR_IMAGE
```

Copy to S3 bucket.

```bash
gcloud storage cp output/gce/image.tar.gz gs://YOUR_BUCKET/gce/image.tar.gz
```

Convert to GCE image.

```bash
gcloud compute images create --project=YOUR_PROJECT \
    --source-uri gs://YOUR_BUCKET/gce/image.tar.gz "custom-cordialsys-treasury-amd64-$(date '+%Y%m%d')" \
    --guest-os-features=GVNIC
```

### For AWS

Build + publish an AMI. Must set the `AWS_*` env variables.

```bash
podman run \
    --rm \
    -it \
    --privileged \
    --pull=newer \
    --security-opt label=disable \
    -v /var/lib/containers/storage:/var/lib/containers/storage \
    -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_SESSION_TOKEN \
    quay.io/centos-bootc/bootc-image-builder:latest \
    --type ami \
    --aws-ami-name custom-cordialsys-treasury-amd64-$(date '+%Y%m%d') \
    --aws-bucket YOUR_S3_BUCKET \
    --aws-region us-east-1 \
    YOUR_IMAGE
```

### Other platforms

Refer to [bootc-image-builder](https://github.com/osbuild/bootc-image-builder) for
how to build images for other platforms or bare-metal.
