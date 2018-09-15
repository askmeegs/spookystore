#!/bin/bash


# Logs 
kubectl get pods | grep "backend-" | cut -d ' ' -f 1 | xargs -I{} kubectl logs -f {}

kubectl get pods | grep "frontend-" | cut -d ' ' -f 1 | xargs -I{} kubectl logs -f {}


# Delete pod 
kubectl get pods | grep "frontend-" | cut -d ' ' -f 1 | xargs -I{} kubectl delete pod {}; kubectl get pods | grep "backend-" | cut -d ' ' -f 1 | xargs -I{} kubectl delete pod {}


