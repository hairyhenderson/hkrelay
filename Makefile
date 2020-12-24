DOCKER_BUILDKIT ?= 1

all: docker

docker: Dockerfile
	@docker buildx build \
		--platform linux/amd64 \
		--platform linux/arm/v6 \
		--platform linux/arm64 \
		--push \
		--tag hairyhenderson/hkrelay .
