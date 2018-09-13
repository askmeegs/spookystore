kubectl create -f web/configmap.yml 
kubectl create -f web/deployment.yml 
kubectl create -f web/network-policy.yml 
kubectl create -f web/service.yml 

kubectl create -f users/configmap.yml 
kubectl create -f users/deployment.yml 
kubectl create -f users/network-policy.yml 
kubectl create -f users/service.yml 

gcloud compute addresses create web2-ip --region us-central1
