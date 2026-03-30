# Tool versions
CONTROLLER_GEN_VERSION ?= v0.15.0

.PHONY: generate manifests run docker-build

## generate: Run controller-gen to regenerate DeepCopy methods
generate:
	go run sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION) \
		object:headerFile="hack/boilerplate.go.txt" paths="./..."

## manifests: Generate CRD YAML from Go types
manifests:
	go run sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION) \
		crd paths="./..." output:crd:artifacts:config=config/crd

## run: Run the controller locally (requires a kubeconfig pointing to a cluster)
run:
	go run ./main.go

## docker-build: Build the controller Docker image
docker-build:
	docker build -t greeting-controller:latest .

## install: Apply the CRD to the current cluster
install:
	kubectl apply -f config/crd/

## sample: Apply the sample Greeting resource
sample:
	kubectl apply -f config/samples/

## logs: Tail controller logs (if running in-cluster)
logs:
	kubectl logs -l app=greeting-controller -f
