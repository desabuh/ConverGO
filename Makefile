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




define RUN_PEER_DETATCHED
	@docker run -d \
		--name $(1) \
		--network $(NETWORK) \
		-e DNS=$(1) \
		-e COMMAND_FILE=/$(TEST_DIR)/$(1).txt \
		-v $(PROJECT_DIR)/$(TEST_DIR)/$(1).txt:/$(TEST_DIR)/$(1).txt \
		$(IMAGE)
endef

#pass automatic variable peer$(N) as $(1) and invoke RUN_PEER_DETATCHED
$(PEERS):
	$(call RUN_PEER_DETATCHED,$@)

# peer1:
# 	docker run -d \
# 		--name peer1 \
# 		--network $(NETWORK) \
# 		-e DNS=peer1 \
# 		-e COMMAND_FILE=/test_domain_concurrent/peer1.txt \
# 		-v $(PROJECT_DIR)/test_domain_concurrent/peer1.txt:/test_domain_concurrent/peer1.txt \
# 		$(IMAGE)

# peer2:
# 	docker run -d \
# 		--name peer2 \
# 		--network $(NETWORK) \
# 		-e DNS=peer2 \
# 		-e COMMAND_FILE=/test_domain_concurrent/peer2.txt \
# 		-v $(PROJECT_DIR)/test_domain_concurrent/peer2.txt:/test_domain_concurrent/peer2.txt \
# 		$(IMAGE)




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
		-e DNS=$(1) \
		$(IMAGE)
endef


dock-test-dev-int: dock-clean dock-build-dev dock-net
	$(call RUN_PEER,$(NAME_INT))

dock-test-int: dock-clean dock-build dock-net
	$(call RUN_PEER,$(NAME_INT))


# wait:
# 	docker wait peer1
# 	docker wait peer2

#$$ is turnaround to avoid make conflict
wait:
	@for p in $(PEERS); do docker wait $$p > /dev/null; done

# verify:
# 	@docker cp peer1:/bob/file.txt peer1.txt
# 	@docker cp peer2:/bob/file.txt peer2.txt

# 	@cmp -s peer1.txt peer2.txt && ( \
# 		echo ============================== && \
# 		echo SUCCESS: peers converged && \
# 		echo ------------------------------ && \
# 		cat peer1.txt && \
# 		echo && \
# 		echo ============================== \
# 	) || ( \
# 		echo ============================== && \
# 		echo FAILURE: peers diverged && \
# 		echo ------------------------------ && \
# 		echo PEER1: && \
# 		cat peer1.txt && \
# 		echo && \
# 		echo ------------------------------ && \
# 		echo PEER2: && \
# 		cat peer2.txt && \
# 		echo && \
# 		echo ============================== \
# 	)

DOMAIN_SERVER_PATH = /master/

verify:

	@for p in $(PEERS); do \
		if ! docker cp $$p:$(DOMAIN_SERVER_PATH)/file.txt $$p.txt 2>/dev/null; then \
			echo "[ERROR] $$p did not write on its own $$p:$(DOMAIN_SERVER_PATH)/file.txt, cannot test convergence"; \
			exit; \
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




# define DOCK_DUMP
# 	@for p in $(PEERS); do \
# 		echo ==============================; \
# 		echo ============$$p=============; \
# 		echo; \
# 		$(1); \
# 		echo; \
# 	done
# endef

# dock-dump:
# 	$(call DOCK_DUMP,docker logs $$p)

dock-dump:
	@docker logs $(PEER_NAME)

dock-dump-file:
	@rm -rf $(PROJECT_DIR)/$(TEST_DIR)/dump
	@mkdir -p $(PROJECT_DIR)/$(TEST_DIR)/dump
	@for p in $(PEERS); do \
		docker logs $$p > $(PROJECT_DIR)/$(TEST_DIR)/dump/$$p.log; \
	done


dock-test-image: dock-net $(PEERS)

