kubectl delete -f web/configmap.yml 
kubectl delete -f web/deployment.yml 
kubectl delete -f web/network-policy.yml 
kubectl delete -f web/service.yml 

kubectl delete -f users/configmap.yml 
kubectl delete -f users/deployment.yml 
kubectl delete -f users/network-policy.yml 
kubectl delete -f users/service.yml 

gcloud compute addresses delete web2-ip --region us-central1
