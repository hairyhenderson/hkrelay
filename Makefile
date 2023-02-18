DOCKER_BUILDKIT ?= 1

all: docker

# Add --platform linux/arm/v6 to build for 32-bit Pi

docker: Dockerfile
	@docker buildx build \
		--platform linux/amd64 \
		--platform linux/arm64 \
		--push \
		--tag hairyhenderson/hkrelay .
