.PHONY: build

build:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build


invoke-callback:
	sam local invoke CallbackFunction -e events/event_org.json --env-vars environmental_vars.json
run-cb: build invoke-callback

build-init:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build InitialAuthorizationFunction
invoke-init:
	sam local invoke InitialAuthorizationFunction -e events/event_init.json --env-vars environmental_vars.json

run-init: build-init invoke-init
