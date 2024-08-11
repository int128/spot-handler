PROJECT_NAME := spot-handler
CLUSTER_NAME := $(PROJECT_NAME)-e2e
KUBECONFIG := output/kubeconfig.yaml
export KUBECONFIG

all:

cluster: $(KUBECONFIG)
$(KUBECONFIG):
	kind create cluster --name $(CLUSTER_NAME)
	kubectl version

delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)
	-rm $(KUBECONFIG)

sqs:
	docker run -d --name $(PROJECT_NAME)-sqs -p 4566:4566 localstack/localstack:3.6.0
	AWS_ENDPOINT_URL=http://localhost:4566 AWS_REGION=us-west-2 AWS_ACCESS_KEY_ID=x AWS_SECRET_ACCESS_KEY=x \
		aws sqs create-queue --queue-name spot-handler-e2e

delete-sqs:
	-docker rm -f $(PROJECT_NAME)-sqs

deploy:
	cd controller && kustomize edit set image controller="$(CONTROLLER_IMAGE)"
	kustomize build controller | kubectl apply -f -
	kubectl -n $(PROJECT_NAME)-system rollout status deployment $(PROJECT_NAME)-controller-manager
	kubectl -n $(PROJECT_NAME)-system get deployment
	kubectl apply -f fixtures/queue.yaml
	kubectl apply -f fixtures/node.yaml

test:
	AWS_ENDPOINT_URL=http://localhost:4566 AWS_REGION=us-west-2 AWS_ACCESS_KEY_ID=x AWS_SECRET_ACCESS_KEY=x \
		aws sqs send-message --queue-url http://localhost:4566/000000000000/spot-handler-e2e \
		--message-body "$$(cat fixtures/message.json)"
	until kubectl get spotinterruptednode fixture-node; do sleep 1; done

logs-controller:
	-kubectl -n $(PROJECT_NAME)-system logs -l control-plane=controller-manager --all-containers --tail=-1
