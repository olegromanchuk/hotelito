.PHONY: build

build-all:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build

build-cb:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build CallbackFunction

invoke-cb:
	sam local invoke CallbackFunction -e events/event_org.json --env-vars environmental_vars.json
run-cb: build-cb invoke-cb

build-init:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build InitialAuthorizationFunction
invoke-init:
	sam local invoke InitialAuthorizationFunction -e events/event_init.json --env-vars environmental_vars.json
run-init: build-init invoke-init

build-3cxlookup:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build 3CXLookupByNumberFunction
invoke-3cxlookup:
	sam local invoke 3CXLookupByNumberFunction -e events/event_lookup.json --env-vars environmental_vars.json
run-lookup: build-3cxlookup invoke-3cxlookup

build-3cxcall:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build 3CXOutboundCallFunction
invoke-3cxcall:
	sam local invoke 3CXOutboundCallFunction -e events/event_3cx_call.json --env-vars environmental_vars.json
run-3cxcall: build-3cxcall invoke-3cxcall

install:
	#create roomid_map.json in S3 bucket