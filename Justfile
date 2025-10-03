
dev:
    GOWORK=off gow run -v ./cmd/panel/... start --treasury-home ./data/treasury --binary-dir ./data/bin --panel-dir ./data/panel 

semver := `date -u +%Y.%m.%d.%H`
ldflags := \
        " -X github.com/cordialsys/panel/pkg/version.Version=" + semver
BUILD_FLAGS := "-ldflags " + "'" + ldflags + " '"

install:
    GOWORK=off go install -v {{BUILD_FLAGS}} ./cmd/panel/...

test:
    GOWORK=off go test -cover ./...

lint:
        GOWORK=off go vet ./...
        GOWORK=off go fmt ./...

tidy:
    # keep gowork off as we want panel to be split into separate repo eventually.
    GOWORK=off go mod tidy

build-dev:
    docker buildx bake --set "*.platform=linux/$(uname -m)" dev

run-dev-1:
    docker run --privileged --name vm-panel-1 --hostname vm-panel-1-dev-${USER} --rm -ti \
        -v base-etc-1:/etc \
        -v base-var-1:/var \
        -v $(pwd)/../:/src:ro \
        -e DEPLOYMENT=dev \
        -e DEVELOPMENT=1 \
        -e HOST_OS=$(uname -o) \
        -p 7661:7666 \
        panel-base

run-dev-2:
    docker run --privileged --name vm-panel-2 --hostname vm-panel-2-dev-${USER} --rm -ti \
        -v base-etc-2:/etc \
        -v base-var-2:/var \
        -v $(pwd)/../:/src:ro \
        -e DEPLOYMENT=dev \
        -e DEVELOPMENT=1 \
        -e HOST_OS=$(uname -o) \
        -p 7662:7666 \
        panel-base

rm:
    docker rm -f vm-panel-1 vm-panel-2 vm-panel

exec:
    docker exec -it vm-panel-1 /bin/zsh
exec-1:
    docker exec -it vm-panel-1 /bin/zsh
exec-2:
    docker exec -it vm-panel-2 /bin/zsh

dev-container n="1":
    docker exec -it --workdir /src/panel -e GOWORK=off vm-panel-{{n}} \
    gow run -v ./cmd/panel/... start  -l 0.0.0.0:7666

setup-download n="1":
    curl -X PUT -d "$TREASURY_API_KEY" -v localhost:766{{n}}/v1/panel/api-key; echo
    curl -X POST localhost:766{{n}}/v1/binaries/cord/versions/latest/install; echo
    curl -X POST localhost:766{{n}}/v1/binaries/treasury-cli/versions/latest/install; echo
    curl -X POST localhost:766{{n}}/v1/binaries/signer/versions/latest/install; echo

activate api_key n="1" :
    docker exec -it --workdir /src/panel vm-panel-{{n}} \
    panel activate --api-key {{api_key}} --version preview

activate-api-key api_key n="1" :
    curl -X POST localhost:766{{n}}/v1/activate/api-key -d '{"api_key":"{{api_key}}"}'; echo

activate-binaries n="1" :
    curl -X POST localhost:766{{n}}/v1/activate/binaries; echo

activate-network n="1":
    curl -X POST localhost:766{{n}}/v1/activate/network; echo

activate-backup n="1":
    curl -X POST localhost:766{{n}}/v1/activate/backup -d '{"baks":[{"bak":"age1uyqfx64fl6g65usx384lq5nrd4c7lsru5n9rsx7kvlsn2lrt6qms9s4vs6"}]}'; echo

generate-treasury n="1":
    curl -X POST localhost:766{{n}}/v1/treasury ; echo

delete-treasury n="1":
    curl -X DELETE localhost:766{{n}}/v1/treasury ; echo

complete-treasury n="1":
    curl -X POST localhost:766{{n}}/v1/treasury/complete ; echo

use-image n="1":
    curl -X POST localhost:766{{n}}/v1/treasury/image -d '{"image":"us-docker.pkg.dev/cordialsys/containers/treasury:25.9.2"}' ; echo

services-list n="1":
    curl -X GET localhost:766{{n}}/v1/services | jq; echo

service service action n="1" :
    curl -X POST localhost:766{{n}}/v1/services/{{service}}/{{action}} ; echo

healthy n="1" :
    curl -X GET 'localhost:766{{n}}/v1/treasury/healthy?verbose' ; echo

overwrite-panel n="1":
    docker exec -it --workdir /src vm-panel-{{n}} \
        bash -c "just panel/ install && cp -f /var/home/root/go/bin/panel /usr/bin/panel"
    docker exec -it --workdir /src vm-panel-{{n}} \
        bash -c "systemctl restart panel"

delete-treasuries:
    docker exec -it --workdir /src vm-panel-1 \
        bash -c "panel delete-treasury"
    docker exec -it --workdir /src vm-panel-2 \
        bash -c "panel delete-treasury"

sync-peers n="1":
    docker exec -it --workdir /src vm-panel-{{n}} \
        bash -c "panel sync-peers"

# normally the VM will have the binaries, or `setup-download` will do this.
# But if there's unreleased features we want to rely on, need to compile in VM and then copy in.
overwrite-binaries n="1":
    docker exec -it --workdir /src vm-panel-{{n}} \
        just cord/ install
    docker exec -it --workdir /src vm-panel-{{n}} \
        mv /var/home/root/go/bin/cord /usr/bin/cord

    docker exec -it --workdir /src \
        -e CARGO_TARGET_DIR=/var/home/root/.cargo/target \
        vm-panel-{{n}} \
        just signer/ install
    docker exec -it --workdir /src vm-panel-{{n}} \
        mv /var/home/root/.cargo/bin/signer /usr/bin/signer

api:
    curl https://api.stoplight.io/projects/cHJqOjIzOTcxNQ/branches/main/export/reference/admin.yaml > docs/admin.yaml
    oapi-codegen -config oapi-codegen.yaml -package admin docs/admin.yaml | grep -v WARNING > pkg/admin/api.gen.go