.PHONY: build run test

#needed to avoid windows MYSYS path automtic conversion issues when running docker commands from makefile
export MSYS_NO_PATHCONV=1
export MSYS2_ARG_CONV_EXCL=*

PROJECT_DIR := $(CURDIR)/test_convergent

build:
	@go build -o bin/main 

run: build
	@./bin/main

test:
	go clean -testcache && go test -v ./...


IMAGE=peer
NETWORK=convergo-net

dock-build:
	@docker build --no-cache --tag $(IMAGE) .

dock-build-dev:
	@docker build --target dev-release-stage --tag $(IMAGE) .

dock-net:
	docker network create $(NETWORK) || true

#change this to support larger number of peers
MAX_PEERS_NUM = 10



# generate peer1, peer2, ..., peerN targets based on the value of first arg
define PEER_SQ
	$(shell seq 1 $(1) | sed 's/.*/peer&/')
endef


PEERS = $(call PEER_SQ, $(N))

MAX_PEERS = $(call PEER_SQ, $(MAX_PEERS_NUM))


DOMAIN_SERVER_PATH = /master/
TARGET_FILE = file.txt

define RUN_PEER_DETATCHED
	@docker run -d \
		--name $(1) \
		--network $(NETWORK) \
		--env-file $(PROJECT_DIR)/$(TEST_DIR)/.env/global.env \
		--env-file $(PROJECT_DIR)/$(TEST_DIR)/.env/$(1).env \
		-e DOMAIN=$(DOMAIN_SERVER_PATH) \
		-e TARGET_FILE=$(TARGET_FILE) \
		-e COMMAND_FILE=/$(TEST_DIR)/$(1).txt \
		-v $(PROJECT_DIR)/$(TEST_DIR)/$(1).txt:/$(TEST_DIR)/$(1).txt \
		$(IMAGE)
endef

#pass automatic variable peer$(N) as $(1) and invoke RUN_PEER_DETATCHED
$(PEERS):
	$(call RUN_PEER_DETATCHED,$@)





dock-clean:
	@docker image rm -f $(IMAGE) || true
	@docker rm -f $(MAX_PEERS)
	@docker network rm $(NETWORK) || true


dock-test: dock-clean dock-build dock-net $(PEERS) wait verify dock-dump-file

dock-test-dev: dock-clean dock-build-dev dock-net $(PEERS) wait verify dock-dump-file


define RUN_PEER
	docker run --rm -it \
		--name $(1) \
		--network $(NETWORK) \
		$(IMAGE)
endef


dock-test-dev-int: dock-clean dock-build-dev dock-net
	$(call RUN_PEER,$(NAME_INT))

dock-test-int: dock-clean dock-build dock-net
	$(call RUN_PEER,$(NAME_INT))



#$$ is turnaround to avoid make conflict when invoking bash commands
wait:
	@for p in $(PEERS); do docker wait $$p > /dev/null; done



verify:

	@for p in $(PEERS); do \
		if ! docker cp $$p:$(DOMAIN_SERVER_PATH)/file.txt $$p.txt 2>/dev/null; then \
			echo "[ERROR] $$p did not write on its own $$p:$(DOMAIN_SERVER_PATH)/file.txt, cannot test convergence" > $$p.txt; \
		fi; \
	done; \
	set -- $(PEERS); base=$$1; \
	all_ok=true; \
	for p in $(PEERS); do \
		if ! cmp -s $$base.txt $$p.txt; then \
			all_ok=false; \
		fi \
	done; \
	if $$all_ok; then \
		echo ==============================; \
		echo SUCCESS: peers converged; \
		echo ------------------------------; \
		cat $$base.txt; \
		echo; \
		echo ==============================; \
	else \
		echo ==============================; \
		echo FAILURE: peers diverged; \
		echo ------------------------------; \
		for p in $(PEERS); do \
			echo $$p:; \
			cat $$p.txt; \
			echo; \
		done; \
		echo ==============================; \
	fi; \
	rm -f $(addsuffix .txt,$(PEERS))



dock-dump:
	@docker logs -t $(PEER_NAME)

dock-dump-file:
	@rm -rf $(PROJECT_DIR)/$(TEST_DIR)/dump
	@mkdir -p $(PROJECT_DIR)/$(TEST_DIR)/dump
	@for p in $(PEERS); do \
		docker logs -t $$p > $(PROJECT_DIR)/$(TEST_DIR)/dump/$$p.log; \
	done



