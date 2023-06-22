# cloudbeds-3cx-integration
Cloudbeds-3CX integration-A009


## Deploy

### Local testing
cd cloudbeds/
env GOOS=linux go build -o cloudbeds
cd ../
sam local start-api