[![build](https://github.com/olegromanchuk/hotelito/actions/workflows/ci.yml/badge.svg)](https://github.com/olegromanchuk/hotelito/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/olegromanchuk/hotelito/badge.svg?branch=master)](https://coveralls.io/github/olegromanchuk/hotelito?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/olegromanchuk/hotelito)](https://goreportcard.com/report/github.com/olegromanchuk/hotelito)
[![Go Reference](https://pkg.go.dev/badge/github.com/olegromanchuk/hotelito.svg)](https://pkg.go.dev/github.com/olegromanchuk/hotelito)
[![GitHub release (release name instead of tag name)](https://img.shields.io/github/v/release/olegromanchuk/hotelito)](https://github.com/olegromanchuk/hotelito/releases)

# Hotelito


Hotelito is an integration app between PBX and hospitality. The current version supports 3CX PBX and Cloudbeds only, but the project was designed to easily plug-in other systems from both ends.


## Supported systems:


PBX:
- 3CX [https://www.3cx.com](https://www.3cx.com)


Hospitality:
- Cloudbeds [https://www.cloudbeds.com](https://www.cloudbeds.com)


## Features
- maid service (updates housekeeping status in hospitality software if a call to a particular extension is placed from a hotel room). Currently supported codes:
* 501 - "clean"
* 502 - "dirty"  

The system can recognize the room from which the call was placed and according to the code, set the room's status accordingly.



## System-specific information (Cloudbeds-3CX)
### General description
Each hotel room has its own phone with an extension. When the room is cleaned (or inspected) it is possible to update the status of the room from the room phone by dialing specific feature codes. These codes are programmed on 3CX. Also, it is possible to pass a maid identifier by assigning different codes to different people: for example:
* "Maid GREEN" will have codes: 501 (clean), and 502 (dirty)
* "Maid BLUE" will have codes: 521 (clean), and 522 (dirty)
and so on. This is one of the most accessible options for achieving the result. The other possible option would be to enter the maidID via DTMF, but it is not implemented yet.



## Getting Started
You can install the integration as:
- AWS lambda function on AWS
- on a dedicated server (valid public https is required)
- as a standalone app installed directly on 3CX (not recommended)

### Prerequisites before app installation
#### Cloudbeds
The Cloudbeds platform supports REST API integration. You need to enable [REST API](https://integrations.cloudbeds.com/hc/en-us/articles/360012140013-Property-and-Group-Account-API-Access) to be able to use this integration.
Note, that the server with the app should have a public **valid** HTTPS endpoint to be able to be authenticated on Cloudbeds via OAuth2.

1. Get Cloudbeds API credentials. Make sure that you select a proper permission scope  
`read:reservation,write:reservation,read:room,write:room,read:housekeeping,write:housekeeping,read:item,write:item`  
 
2. SKIP it for AWS lambda version.  
Set a correct redirect URL. It should be  
`https://mypublic.api.address/api/v1/callback`

3. You will need to create a configuration file with the credentials (.env). Check .env_example for the list of required variables.  
   PS. `APPLICATION_NAME` will be used in AWS Parameter Store path (if AWS lambda version is used).

4. `CLOUDBEDS_REDIRECT_URL` should be set to the public IP address of the server plus "/api/v1/callback". On this URL Cloudbeds authentication server will send an authorization code as part of the authentication process [OAuth2](https://integrations.cloudbeds.com/hc/en-us/articles/360006450433-OAuth-2-0).

5. Install the app.

### Install standalone version

- Install Hotelito by downloading the latest release from the [Releases](https://github.com/olegromanchuk/hotelito/releases) page.
- Create .env file that will contain all the configuration parameters. See included .env_example.  
Notes on .env file:   
  * CLOUDBEDS_AUTH_URL and CLOUDBEDS_TOKEN_URL are Cloudbeds endpoints. They should not be changed unless Cloudbeds changes them.
  * all parameters started from "AWS" could be ignored for standalone version.
  * LOG_LEVEL: [Trace, Debug, Info, Warning, Error, Fatal, Panic]
- Create roomid_map.json file that will contain the list of room ID's and their extensions. See included roomid_map.json.
For more details check [GH-15](https://github.com/olegromanchuk/hotelito/issues/15)
- Run `make all` to build the app.

### Install AWS Lambda version

- Install AWS [cli](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) and AWS SAM [cli](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html).
- [Configure](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html) AWS cli with your credentials
- Install [Go 1.20.X](https://golang.org/doc/install)
- Create .env file that contains all the configuration parameters. See included .env_example.  
Notes on .env file:
    * CLOUDBEDS_AUTH_URL and CLOUDBEDS_TOKEN_URL are Cloudbeds endpoints on 07/2023. They should not be changed unless Cloudbeds changes them.
    * CLOUDBEDS_REDIRECT_URL will be determined after lambda installation. So, you can leave it as it is.
    * PORT parameter is not used in AWS lambda version.
    * LOG_LEVEL: [Trace, Debug, Info, Warning, Error, Fatal, Panic]
- `cd cmd/hotelito-aws-lambda`
- Run `sam deploy --guided --profile MYPROFILE --no-execute-changeset`. This action is needed to create samconfig.toml file that will be used by the installation script. [More details on how `sam deploy` works](#how-sam-deploy-works)
Notes on `sam deploy guided --profile MYPROFILE --no-execute-changeset`:
    * leave `SAM configuration file [samconfig.toml]:` in default value: *samconfig.toml*
    * Allow SAM CLI IAM role creation: Y
    * Save arguments to samconfig.toml: Y
- Run `make deploy`. This script will deploy the application.
- Set the correct redirect URL in Cloudbeds portal. It will be outputted by the script as "REDIRECT URL". It should be something like `https://deadbeef.execute-api.us-east-1.amazonaws.com/Prod/api/v1/callback`

#### Troubleshooting
If something went wrong - delete samconfig.toml file and run `sam deploy --guided --no-execute-changeset` again.


#### 3CX
3CX does not have REST API. The integration is implemented via a custom CRM integration template.  
The template will be generated by the script during the installation and could be found in the in 3cx/crm-template-cloudbeds-3cx.xml

5. In 3CX admin interface under Settings->(Integrations) CRM click add and select crm-template-cloudbeds-3cx.xml.
   **Important**: when updating the template in 3CX you need to follow the next steps:
- save it;
- then open, disable Call Journaling and save;
- then open again, enable Call Journaling and save.  
  It is needed to clear 3CX caching. Was discovered through numerous tests. If you just add/save a new template the old cached settings will be used.

6. Create IVR "clean" and "dirty" recordings.
7. Setup routing.


#### !!! Important note for AWS Lambda version !!!
If you decide to remove the app from AWS Lambda, make sure to remove all the parameters from AWS Parameter Store. Otherwise, they will be left there and will be accessible to anyone who has access to your AWS account.
All other components will be removed automatically as soon as you remove stack from CloudFormation.


### Helpful links:
#### Cloudbeds
* [Dev documentation](https://integrations.cloudbeds.com/hc/en-us)
* [API reference](https://integrations.cloudbeds.com/hc/en-us/categories/14018007083163-API-Reference)
* [API-list of functions](https://hotels.cloudbeds.com/api/docs/)
* [Login to portal](https://hotels.cloudbeds.com/)
* [Postman Collection](https://app.getpostman.com/run-collection/0f613eb0e2a6a4fff0e9)
* [PBX Integration example](https://integrations.cloudbeds.com/hc/en-us/articles/7147099928859-App-Integration-PBX-Hotspot-TV-And-other-Systems-)

#### 3CX
* [3CX CRM Template Description](https://www.3cx.com/docs/server-side-crm-template-xml-description/)
* [CRM Integration Wizard](https://www.3cx.com/docs/crm-integration/)




## Development notes and testing

Standalone version located in cmd/hotelito. AWS Lambda version located in cmd/hotelito-aws-lambda.
```
├── cmd
│   ├── hotelito
│       └── main.go
│   └── hotelito-aws-lambda
│       ├── ...
```

Shared code is located in `internal` and `pkg` directories.

### Local testing standalone on localhost
```
go build -o cmd/hotelito/hotelito cmd/hotelito/main.go
echo "make sure that .env file is present in the current directory and contains all the required variables"
./cmd/hotelito/hotelito
```

### Local testing standalone in docker
```
 docker build -t hotelito .
 docker run --name hotelito -p 8080:8080 hotelito
```

### Local testing AWS
```
cd app/
make build
sam local invoke HotelitoFunction -e events/event_org.json --env-vars environmental_vars.json

## option 2
make build
sam local start-api -e events/event_org.json --env-vars environmental_vars.json
```
`sam build` creates .aws-sam directory that is used for `sam local start-api`. Keep that in mind when running `sam local start-api`. If this directory doesn't exist the binary should exist in the directory when a source code is located. You MUST build the binary for Linux, as shown above. If the binary doesn't exist or was built for different architecture you will get an unclear error from sam.  
`sam local generate-event apigateway aws-proxy --method POST --body '{"Number": "2222222501", "CallType": "Outbound", "CallDirection": "Outbound", "Name": "ExampleName", "Agent": "501", "AgentFirstName": "ExampleAgentFirstName", "DateTime": "2023-07-07T14:15:22Z"}' --path '3cx/outbound_call' > events/event_3cx_call.json`

### How sam deploy works
`sam deploy --guided` when run first time creates CF script that creates sam S3 buckets for uploading lambda. Also, it creates a samconfig.toml file. If you set custom profile with `sam deploy --guided --profile MY_AWS_PROFILE`, then MY_AWS_PROFILE will appear in samconfig.toml as 
```
"profile"=MYPROFILE
```
deploy_aws.sh will check on this line and if it is not set will propose to use "default" profile.


##### Consideration about aws parameter store:
The decision to create parameter store variables in a bash script "deploy aws", not in a template.yaml was made because of the following reason: AWS does not support creating secureString in template.yml

### New function
To add new function follow the next steps:
1. create a file with handler function in proper directory
2. add section to template.yaml. Set function name (3CXOutboundCallFunction) , CodeUri, Events->Properties->Path, Events->Properties->Method
3. add new section in sync_environmental_vars.sh. Search for the section called "All functions must be added here"


## TODO
[x] makefile
[x] workflows
[x] TODO moved to GH Issues
[x] workflows
