.PHONY: build

# samconfig.toml is generated automatically after run 'sam deploy...' Check README.md for details
AWS_CONFIG_PROFILE=$$(grep 'profile' ./samconfig.toml | awk -F '"' '{print $$2}' | cut -d'"' -f 1)

build-all:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build

build-cb:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build CallbackFunction

invoke-cb:
	sam local invoke --profile $(AWS_CONFIG_PROFILE) CallbackFunction -e events/events_oauthcallback.json --env-vars environmental_vars.json
debug-cb:
	sam local invoke --profile $(AWS_CONFIG_PROFILE) -d 5986 CallbackFunction -e events/events_oauthcallback.json --debugger-path=/Users/xxot/go/bin/linux_amd64 --env-vars environmental_vars.json --debug-args="-delveAPI=2"
run-cb: build-cb invoke-cb

build-init:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build InitialAuthorizationFunction
invoke-init:
	sam local invoke --profile $(AWS_CONFIG_PROFILE) InitialAuthorizationFunction -e events/event_init.json --env-vars environmental_vars.json
debug-init:
	sam local invoke --profile $(AWS_CONFIG_PROFILE) -d 5986 InitialAuthorizationFunction -e events/event_init.json --debugger-path=/Users/xxot/go/bin/linux_amd64 --env-vars environmental_vars.json --debug-args="-delveAPI=2"
run-init: build-init invoke-init

build-3cxlookup:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build 3CXLookupByNumberFunction
invoke-3cxlookup:
	sam local invoke --profile $(AWS_CONFIG_PROFILE) 3CXLookupByNumberFunction -e events/event_lookup.json --env-vars environmental_vars.json
run-lookup: build-3cxlookup invoke-3cxlookup

build-3cxcall:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build 3CXOutboundCallFunction
invoke-3cxcall:
	sam local invoke --profile $(AWS_CONFIG_PROFILE) 3CXOutboundCallFunction -e events/events_3cxoutboundcall.json --env-vars environmental_vars.json
debug-3cxcall:
	sam local invoke --profile $(AWS_CONFIG_PROFILE) -d 5986 3CXOutboundCallFunction -e events/events_3cxoutboundcall.json --debugger-path=/Users/xxot/go/bin/linux_amd64 --env-vars environmental_vars.json --debug-args="-delveAPI=2"
run-3cxcall: build-3cxcall invoke-3cxcall

install:
	#create config.json in S3 bucket
deploy: build-all
	#go test -v ./...
	./deploy_aws.sh
