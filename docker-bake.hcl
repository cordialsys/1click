variable "TAG" {
  default = "latest"
}
variable "BASE" {
  default = "dev"
}
variable "COMMIT" {
  default = "unknown"
}
variable "SEMVER" {
  default = "unknown"
}
variable "platforms" {
  default = ["linux/amd64", "linux/arm64"]
  # default = ["linux/arm64"]
}

target "base-dev" {
  dockerfile = "./Containerfile"
  platforms = platforms
  context = "./infra/vm/dev"
}

target "base-gcp" {
  dockerfile = "./Containerfile"
  platforms = platforms
  context = "./infra/vm/gcp"
}

target "base-aws" {
  dockerfile = "./Containerfile"
  platforms = platforms
  context = "./infra/vm/aws"
}

target "build-panel" {
  dockerfile = "./container.panel.Dockerfile"
  platforms = platforms
  context = "."
}

target "vm" {
  dockerfile = "container.vm.Dockerfile"
  platforms = platforms
  tags = [
  ]
  context = "."
  contexts = {
    base = "target:base-${BASE}"
    build-panel = "target:build-panel"
  }
}

target "dev" {
  dockerfile = "container.dev.Dockerfile"
  platforms = platforms
  tags = [
    "panel-base",
  ]
  context = "."
  contexts = {
    vm = "target:vm"
  }
}

group "default" {
#   targets = ["demo"]
  targets = []
}