SHELL = bash

files/id_ed25519:
	mkdir -p files
	ssh-keygen -t ed25519 -a 64 -N "" -f "$@"

files/config.toml: config.toml
ifndef RUNNER_TOKEN
	$(error RUNNER_TOKEN is not defined)
endif
ifndef HCLOUD_TOKEN
	$(error HCLOUD_TOKEN is not defined)
endif
	mkdir -p files
	envsubst '$$RUNNER_TOKEN,$$HCLOUD_TOKEN' < config.toml > "$@"

up: files/id_ed25519 files/config.toml
	BUILDKIT_PROGRESS=plain docker compose up --build --detach

down:
	docker compose down
	bash destroy.sh

clean: down
	rm -Rf files/
