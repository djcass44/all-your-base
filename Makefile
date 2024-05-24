.PHONY: build-ubi lock-ubi

build-ubi:
	go run main.go build -c tests/fixtures/ubi9_full.yaml --save /tmp/test.tar
	docker load < /tmp/test.tar

lock-ubi:
	go run main.go lock -c tests/fixtures/ubi9_full.yaml