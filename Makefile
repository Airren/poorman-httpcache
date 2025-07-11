.PHONY: build test debug
build:
	go build -o proxy ./cmd/httptrace
	chmod +x httptrace

test:
	go test ./...

debug:
	echo "testing"

deploy: build
	nohup ./httptrace > trace.log 2>&1 & echo $$! > save_pid.txt

kill:
	kill -9 $$(cat save_pid.txt) && rm save_pid.txt