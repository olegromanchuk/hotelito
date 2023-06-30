[![Coverage Status](https://coveralls.io/repos/github/olegromanchuk/hotelito/badge.svg?branch=master)](https://coveralls.io/github/olegromanchuk/hotelito?branch=master)


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
- Lambda function on AWS (preferred option)
- on a dedicated server (better option) (valid public https is required)
- as a standalone app installed directly on 3CX (not recommended)

#### Cloudbeds
The Cloudbeds platform supports REST API integration. You need to enable [REST API](https://integrations.cloudbeds.com/hc/en-us/articles/360012140013-Property-and-Group-Account-API-Access) to be able to use this integration. Another option would be to install the app from the Cloudbeds Marketplace (not implemented yet).  
Note, that the server with the app should have a public **valid** HTTPS endpoint to be able to authenticate on Cloudbeds via OAuth2.

1. Get Cloudbeds API credentials. Make sure that you select a proper permission scope  
`read:reservation,write:reservation,read:room,write:room,read:housekeeping,write:housekeeping,read:item,write:item`  
 Set a correct redirect URL. It should be  
`https://mypublic.api.address/api/v1/callback`

2. You will need to update a configuration file with the credentials (.env)
```
CLOUDBEDS_CLIENT_ID=mycompanyexample_LuPCZsereqdqdXjS
CLOUDBEDS_CLIENT_SECRET=haPpyjHKJujewnfw32SDDFFD
CLOUDBEDS_REDIRECT_URL=https://mypublic.api.address/api/v1/callback
CLOUDBEDS_SCOPES=read:reservation,write:reservation,read:room,write:room,read:housekeeping,write:housekeeping,read:item,write:item
CLOUDBEDS_AUTH_URL=https://hotels.cloudbeds.com/api/v1.1/oauth
CLOUDBEDS_TOKEN_URL=https://hotels.cloudbeds.com/api/v1.1/access_token
```
3. `CLOUDBEDS_REDIRECT_URL` should be set to the public IP address of the server plus "/api/v1/callback". On this URL Cloudbeds authentication server will send an authorization code as part of the authentication process [OAuth2](https://integrations.cloudbeds.com/hc/en-us/articles/360006450433-OAuth-2-0).

#### 3CX
3CX does not have REST API. The integration is implemented via a custom CRM integration template.

4. Prepare crm-template-cloudbeds-3cx.xml Update Url in 3 locations:
```
<Scenarios>
    <Scenario Id="" Type="REST">
      <Request Url="https://bb37-7232-283123 2-10.ngrok-free.app/api/v1/lookupbynumber?Number=[Number]&amp;CallDirection=[CallDirection]" MessagePasses="0" RequestEncoding="UrlEncoded" RequestType="Get" ResponseType="Json" />
```
```
 <Scenario Id="ReportCall" Type="REST">
      <Request SkipIf="[IIf([ReportCallEnabled]!=True||[EntityId]==&quot;&quot;,True,[IIf([CallType]!=Inbound,True,False)])]" Url="https://bb37-7232-283123 2-10.ngrok-free.app/api/v1/3cx/outbound_call" MessagePasses="0" RequestContentType="application/json" RequestEncoding="Json" RequestType="Post" ResponseType="Json">
```
```
 <Scenario Id="ReportCallOutbound" Type="REST">
      <Request SkipIf="[IIf([ReportCallEnabled]!=True||[EntityId]==&quot;&quot;,True,[IIf([CallType]!=Outbound,True,False)])]" Url="https://bb37-7232-283123 2-10.ngrok-free.app/api/v1/3cx/outbound_call" MessagePasses="0" RequestContentType="application/json" RequestEncoding="Json" RequestType="Post" ResponseType="Json">
```

5. In 3CX admin interface under Settings->(Integrations) CRM click add and select crm-template-cloudbeds-3cx.xml.
**Important**: when updating the template in 3CX you need to follow the next steps:
- save it; 
- then open, disable Call Journaling and save; 
- then open again, enable Call Journaling and save.  
It is needed to clear 3CX caching. Was discovered through numerous tests. If you just add/save a new template the old cached settings will be used.

TODO   
6. Create IVR "clean" and "dirty".  


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


## Deploy

### Local testing standalone
cd hotelito
go build -o cmd/hotelito/hotelito cmd/hotelito/main.go
./cmd/hotelito/hotelito

### Local testing AWS
cd cloudbeds/
env GOOS=linux go build -o cloudbeds
cd ../
sam local start-api

## TODO
[ ] makefile
[ ] workflows