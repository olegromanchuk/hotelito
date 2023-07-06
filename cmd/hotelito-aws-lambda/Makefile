.PHONY: build

build:
	./sync_environmental_vars.sh
	rm -rf .aws-sam
	sam build
invoke:
	sam local invoke HotelitoFunction -e events/event_org.json --env-vars environmental_vars.json

run: build invoke
