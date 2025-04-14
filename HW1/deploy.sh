#! /bin/bash

YELLOW='\033[0;33m'
NC='\033[0m'
DOT_INTERVAL=1

function run_with_dots() {
    "$@" &
    PID=$!

    while kill -0 $PID 2>/dev/null; do
        echo -ne "${YELLOW}.${NC}" > /dev/tty
        sleep $DOT_INTERVAL
    done

    echo > /dev/tty
}

echo -e "${YELLOW}Rebuilding docker image${NC}"
eval $(minikube docker-env)
docker build -t custom-logger:local logger/app
eval $(minikube docker-env --unset)
echo

echo -e "${YELLOW}Starting logger${NC}"
kubectl apply -f logger/configmap.yaml
kubectl apply -f logger/deployment.yaml
kubectl apply -f logger/service.yaml
kubectl apply -f logger/combiner.yaml
kubectl apply -f logger/archiver.yaml
echo -ne "${YELLOW}Waiting for logger${NC}"
run_with_dots \
kubectl wait --for=condition=Ready --timeout=15s pod -l app=custom-logger &> /dev/null \
kubectl wait --for=condition=Ready --timeout=15s pod -l app=custom-logger-combiner &> /dev/null
echo

echo -e "${YELLOW}Starting tester${NC}"
kubectl apply -f tester/configmap.yaml
kubectl apply -f tester/pod.yaml
echo -ne "${YELLOW}Waiting for tester${NC}"
run_with_dots \
kubectl wait --for=condition=Ready --timeout=15s pod/tester-pod &> /dev/null
echo

kubectl exec -it tester-pod -- sh /test/test.sh

AGENT_POD="$(kubectl get pods --no-headers -l app=custom-logger-combiner | awk '{print $1}')"
echo -e "${YELLOW}Combined logs${NC} (by $AGENT_POD)"
kubectl logs $AGENT_POD
echo

DOT_INTERVAL=5
echo -ne "${YELLOW}Waiting for archiver${NC}"
run_with_dots sleep 60
echo

ARCHIVER_JOB=$(kubectl get jobs | tail -n1 | awk '{print $1}')
echo -e "${YELLOW}Latest archiver job${NC}"
kubectl logs jobs/$ARCHIVER_JOB
echo