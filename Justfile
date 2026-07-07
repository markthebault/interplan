set dotenv-load

default:
    just --list

fmt:
    gofmt -w .

test:
    go test ./...

build:
    go build -o ./bin/interplan ./cmd/interplan

test-manual-complex-demo: build
    ./scripts/manual-complex-demo.sh

test-manual-complex-demo-no-open: build
    INTERPLAN_NO_OPEN=1 ./scripts/manual-complex-demo.sh

test-manual-poll-complex-demo: build
    sh -c 'INTERPLAN_PORT="$(cat /tmp/interplan-complex-demo.port)" ./bin/interplan poll /tmp/interplan-complex-demo.html --timeout-ms 1000'
