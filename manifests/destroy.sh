kubectl delete -f frontend/deployment.yml 
kubectl delete -f frontend/network-policy.yml 
kubectl delete -f frontend/ingress.yml 

kubectl delete -f backend/configmap.yml 
kubectl delete -f backend/deployment.yml 
kubectl delete -f backend/network-policy.yml 
kubectl delete -f backend/service.yml 

