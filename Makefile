.PHONY: build run test

MSYS_NO_PATHCONV=1
MSYS2_ARG_CONV_EXCL="*"

PROJECT_DIR := $(CURDIR)

build:
	@go build -o bin/main 

run: build
	@./bin/main

test:
	go test -v ./...


IMAGE=peer
NETWORK=convergo-net

dock-build:
	@docker build --tag $(IMAGE) .

dock-build-dev:
	@docker build --target dev-release-stage --tag $(IMAGE) .

dock-net:
	docker network create $(NETWORK) || true

peer1:
	docker run -d \
		--name peer1 \
		--network $(NETWORK) \
		-e PEER_ID=1 \
		-e PEER_NAME=Albert \
		-e LOCAL_PORT=8085 \
		-e DNS=peer1 \
		-e COMMAND_FILE=/test_domain_concurrent/peer1.txt \
		-v $(PROJECT_DIR)/test_domain_concurrent/peer1.txt:/test_domain_concurrent/peer1.txt \
		$(IMAGE)

peer2:
	docker run -d \
		--name peer2 \
		--network $(NETWORK) \
		-e PEER_ID=2 \
		-e PEER_NAME=Bob \
		-e LOCAL_PORT=8086 \
		-e DNS=peer2 \
		-e COMMAND_FILE=/test_domain_concurrent/peer2.txt \
		-v $(PROJECT_DIR)/test_domain_concurrent/peer2.txt:/test_domain_concurrent/peer2.txt \
		$(IMAGE)

dock-clean:
	docker image rm -f $(IMAGE) || true
	docker rm -f peer1 peer2 || true
	docker network rm $(NETWORK) || true


dock-test: dock-clean dock-build dock-net peer2 peer1 wait verify 

dock-test-dev: dock-clean dock-build-dev dock-net peer2 peer1 wait verify


wait:
	docker wait peer1
	docker wait peer2

verify:
	@docker cp peer1:/test_domain_concurrent/file.txt peer1.txt
	@docker cp peer2:/test_domain_concurrent/file.txt peer2.txt

	@cmp -s peer1.txt peer2.txt && ( \
		echo ============================== && \
		echo SUCCESS: peers converged && \
		echo ------------------------------ && \
		cat peer1.txt && \
		echo && \
		echo ============================== \
	) || ( \
		echo ============================== && \
		echo FAILURE: peers diverged && \
		echo ------------------------------ && \
		echo PEER1: && \
		cat peer1.txt && \
		echo && \
		echo ------------------------------ && \
		echo PEER2: && \
		cat peer2.txt && \
		echo && \
		echo ============================== \
	)

dock-test-image: dock-net peer2 peer1

