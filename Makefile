.PHONY: build-ubi lock-ubi test

build-ubi:
	go run main.go build -c tests/fixtures/ubi9_full.yaml --save /tmp/test.tar
	docker load < /tmp/test.tar

lock-ubi:
	go run main.go lock -c tests/fixtures/ubi9_full.yaml

test:
	./tests/test_alpine.sh
	./tests/test_debian.sh
	./tests/test_ubi.sh
