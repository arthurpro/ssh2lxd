mkdir -p build/bin

version=1.0

GOPATH=/tmp/.go CGO_ENABLED=0 \
    go build cmd/ssh2lxd/ssh2lxd.go -ldflags="-X 'ssh2lxd.version=1.0'" -o ./build/bin/ssh2lxd

strip -s ./build/bin/ssh2lxd

fpm -s dir -t rpm -C ./build \
    --name ssh2lxd \
    --version 1.0 \
    --iteration 0 \
    --rpm-dist el8 \
    --category Applications/Internet \
    --url https://localhost \
    --config-files /etc \
    --description "SSH server for LXD containers" \
    --after-install=scripts/install.sh \
    --after-remove=scripts/install.sh \
    .

fpm -s dir -t rpm -C ./build \
    --name ssh2lxd \
    --version 1.0 \
    --iteration 0 \
    --rpm-dist el7 \
    --category Applications/Internet \
    --url https://localhost \
    --config-files /etc \
    --description "SSH server for LXD containers" \
    --after-install=scripts/install.sh \
    --after-remove=scripts/install.sh \
    .
