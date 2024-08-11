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

deploy:
	cd controller && kustomize edit set image controller="$(CONTROLLER_IMAGE)"
	kustomize build controller | kubectl apply -f -
	kubectl -n $(PROJECT_NAME)-system rollout status deployment $(PROJECT_NAME)-controller-manager
	kubectl -n default rollout status deployment localstack
	kubectl -n $(PROJECT_NAME)-system get deployment

test:
	kubectl -n default exec deploy/localstack -- awslocal sqs create-queue --queue-name spot-handler-e2e
	kubectl -n default exec deploy/localstack -- awslocal sqs send-message \
		--queue-url http://localhost:4566/000000000000/spot-handler-e2e \
		--message-body "$$(cat fixtures/message.json)"
	until kubectl get spotinterruptednode fixture-node; do sleep 1; done

logs-controller:
	-kubectl -n $(PROJECT_NAME)-system logs -l control-plane=controller-manager --all-containers --tail=-1
