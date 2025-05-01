#! /bin/bash

YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${YELLOW}Cleaning up${NC}"
AGENT_POD="$(kubectl get pods --no-headers -l app=custom-logger-combiner | awk '{print $1}')"
kubectl delete pod $AGENT_POD
kubectl delete pod tester-pod
kubectl delete service custom-logger-service
kubectl delete deployment custom-logger-deployment
minikube ssh "sudo rm /var/log/custom-logger/app.log"
