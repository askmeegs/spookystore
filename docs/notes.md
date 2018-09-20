
## notes 

### protoc 

```
protoc -I . ./spookystore.proto --go_out=plugins=grpc:.
```


### Run from source

```
./bin/spookystore --addr=:8001 --google-project-id=spookystore-18
```

```
./web -addr=:8000 --spooky-store-addr=:8001 \
    --google-oauth2-config=/Users/mokeefe/spooky-oauth.json \
    --google-project-id=spookystore-18
``` 



### grep commands 

```
# Logs 
kubectl get pods | grep "backend-" | cut -d ' ' -f 1 | xargs -I{} kubectl logs -f {}

kubectl get pods | grep "frontend-" | cut -d ' ' -f 1 | xargs -I{} kubectl logs -f {}


# Delete pod 
kubectl get pods | grep "frontend-" | cut -d ' ' -f 1 | xargs -I{} kubectl delete pod {}; kubectl get pods | grep "backend-" | cut -d ' ' -f 1 | xargs -I{} kubectl delete pod {}
``` 

### skaffold 

```
skaffold dev -f skaffold.yaml
```