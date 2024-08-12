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
	kubectl rollout status deployment localstack
	kubectl get deployment -A

test:
	kubectl apply -f fixtures/
	kubectl exec deploy/localstack -- awslocal --region us-west-2 sqs create-queue --queue-name spot-handler-e2e
	kubectl exec deploy/localstack -- awslocal --region us-west-2 sqs send-message \
		--queue-url http://localhost:4566/000000000000/spot-handler-e2e \
		--message-body "$$(cat spot/message.json)"
	until kubectl get -oyaml spotinterruptednode fixture-node; do sleep 1; done

logs-resources:
	-kubectl get Queue
	-kubectl get SpotInterruption
	-kubectl get SpotInterruptedNode
	-kubectl get SpotInterruptedPod -A

logs-controller:
	-kubectl -n $(PROJECT_NAME)-system logs -l control-plane=controller-manager --all-containers --tail=-1

logs-localstack:
	-kubectl logs -l app.kubernetes.io/name=localstack --all-containers --tail=-1
