kubectl create -f frontend/deployment.yml 
kubectl create -f frontend/network-policy.yml 
kubectl create -f frontend/ingress.yml 

kubectl create -f backend/configmap.yml 
kubectl create -f backend/deployment.yml 
kubectl create -f backend/network-policy.yml 
kubectl create -f backend/service.yml 

