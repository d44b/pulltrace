REGISTRY ?= ghcr.io/d44b
VERSION ?= 0.1.0
AGENT_IMAGE = $(REGISTRY)/pulltrace-agent:$(VERSION)
SERVER_IMAGE = $(REGISTRY)/pulltrace-server:$(VERSION)
NAMESPACE ?= pulltrace
HELM_RELEASE ?= pulltrace
KUBECONTEXT ?= d4b

.PHONY: all build test lint ui docker-build docker-push helm-lint helm-install deploy clean e2e

all: lint test docker-build

## Build
build:
	go build -o bin/pulltrace-agent ./cmd/pulltrace-agent
	go build -o bin/pulltrace-server ./cmd/pulltrace-server

ui:
	cd web && npm ci && npm run build

## Test
test:
	go test ./... -v -race

lint:
	go vet ./...

## Docker
docker-build: docker-build-agent docker-build-server

docker-build-agent:
	docker build -f Dockerfile.agent -t $(AGENT_IMAGE) .

docker-build-server:
	docker build -f Dockerfile.server -t $(SERVER_IMAGE) .

docker-push:
	docker push $(AGENT_IMAGE)
	docker push $(SERVER_IMAGE)

## Helm
helm-lint:
	helm lint charts/pulltrace

helm-template:
	helm template $(HELM_RELEASE) charts/pulltrace --namespace $(NAMESPACE)

helm-install:
	kubectl --context $(KUBECONTEXT) create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl --context $(KUBECONTEXT) apply -f -
	helm upgrade --install $(HELM_RELEASE) charts/pulltrace \
		--namespace $(NAMESPACE) \
		--kube-context $(KUBECONTEXT) \
		--set agent.image.repository=$(REGISTRY)/pulltrace-agent \
		--set agent.image.tag=$(VERSION) \
		--set server.image.repository=$(REGISTRY)/pulltrace-server \
		--set server.image.tag=$(VERSION)

## Deploy (build + push + install)
deploy: docker-build docker-push helm-install

## Verify
verify:
	kubectl --context $(KUBECONTEXT) -n $(NAMESPACE) get pods
	kubectl --context $(KUBECONTEXT) -n $(NAMESPACE) rollout status daemonset/$(HELM_RELEASE)-agent --timeout=120s
	kubectl --context $(KUBECONTEXT) -n $(NAMESPACE) rollout status deployment/$(HELM_RELEASE)-server --timeout=120s

port-forward:
	kubectl --context $(KUBECONTEXT) -n $(NAMESPACE) port-forward svc/$(HELM_RELEASE)-server 8080:8080

## E2E
e2e:
	./hack/e2e-test.sh

## Clean
clean:
	rm -rf bin/ web/dist/
	helm uninstall $(HELM_RELEASE) --namespace $(NAMESPACE) --kube-context $(KUBECONTEXT) 2>/dev/null || true
