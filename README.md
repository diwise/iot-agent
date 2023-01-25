# iot-agent
A service that handles (decodes and converts) incoming data from multiple sources.

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://github.com/diwise/iot-agent/blob/main/LICENSE)

# Design
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" width="612px" height="209px" viewBox="-0.5 -0.5 612 209" content="&lt;mxfile host=&quot;app.diagrams.net&quot; modified=&quot;2023-01-25T20:09:39.237Z&quot; agent=&quot;5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36&quot; etag=&quot;BACj9fB5V78Uu0QgB52i&quot; version=&quot;20.8.11&quot;&gt;&lt;diagram name=&quot;Sida-1&quot; id=&quot;caz-yZAls0l2qeqZ1_Nw&quot;&gt;zVlbb5swFP41PLbC2EB4bNOsk6ZJldqp3d5ccIIlwMw4TbJfP1Nsbs6FtrThKZzjY2N/5zsXEwvO0+0tx3n8k0UksRw72lrwxnIcALyZ/Ck1u0rjBbBSrDiNlFGjuKf/iFLaSrumESk6hoKxRNC8qwxZlpFQdHSYc7bpmi1Z0n1rjlfEUNyHODG1jzQScaWdOX6j/07oKhb1gYNqJMXaWJ2kiHHENi0VXFhwzhkT1VO6nZOkBE/jUs37dmC03hgnmRgy4cfD7SKYbaLgCt095sHzH3/NL5QzXnCyVgeOcRYlhKs9i50GgrN1FpFyLWDB601MBbnPcViObqTrpS4WaaKGORNYUJZJ8SKwpUK9hnBBtgf3D2pUJJ0IS4ngO2miJ7iomqKY5CAF7KbxC/CULm75BCodVlRY1Us3aMkHBdgbwPMM8HBOJwhcMDXgfAO49K8QBnIki67KEJZSmOCioGEfLI1siZREg++e2sLvUrh0tXizbQ/e7JRUvZZERh7o4SsTD+YrIo6cC/j7HdEC2t2Ds9Zxkkjnv3T3sQ989YY7RuUOGz/rTKP97LrdJQq25iFRs9qZor8Q6i2kc5peqALCWOiVC/Wx30+PmUGPM1GDbKl4aj23ZkmpmVQKb6dT5Y7Tyfkk7bxzss7pkQV6vawxlHXOrLsQcnoLHWCd5AHetczy0qA4smHYTYfQt4/vyz9qLx+qHYwaAoERAksc4ohMr7rUVWIy1UW3ji3wIvJCJSZlV5bJji0lmVltzo4ksr2pIQn2IBnKHn+C7SFCU+tygGOgJ28pr4ecIn4zMDX8kIHf7eKhvEvqaJZwLunqo82BLvKXvtuu80erfNNRqFm6qQCjdZsn2wOd5072B8FZ+4N+WX9vfwB7hQaBYf3BWCUZuAYd735dvzIPi3UxGgvbJARDSdjhoP11HAwGUhDY1hk5CNEJ6gzlIPIPNDtfxUHz5vyxSrKkSTJnCeNSzlhGRuoJ7S5MwDdrCQz2+Ls2HL+YjH+pVBELrOGXw8//RnG6avhDQ/asZQP1r5XBO0PW69Uf46vIZ4eseZWjTFyEjH/wMvc5oev1bsf7QnesyJVi8/m7wrv5EwEu/gM=&lt;/diagram&gt;&lt;/mxfile&gt;"><defs/><g><rect x="41" y="70" width="160" height="30" rx="4.5" ry="4.5" fill="rgb(255, 255, 255)" stroke="rgb(0, 0, 0)" transform="rotate(-90,121,85)" pointer-events="all"/><g transform="translate(-0.5 -0.5)rotate(-90 121 85)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 158px; height: 1px; padding-top: 85px; margin-left: 42px;"><div data-drawio-colors="color: rgb(0, 0, 0); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; white-space: normal; overflow-wrap: normal;">handler</div></div></div></foreignObject><text x="121" y="89" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="12px" text-anchor="middle">handler</text></switch></g><rect x="81" y="70" width="160" height="30" rx="4.5" ry="4.5" fill="rgb(255, 255, 255)" stroke="rgb(0, 0, 0)" transform="rotate(-90,161,85)" pointer-events="all"/><g transform="translate(-0.5 -0.5)rotate(-90 161 85)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 158px; height: 1px; padding-top: 85px; margin-left: 82px;"><div data-drawio-colors="color: rgb(0, 0, 0); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; white-space: normal; overflow-wrap: normal;">api</div></div></div></foreignObject><text x="161" y="89" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="12px" text-anchor="middle">api</text></switch></g><path d="M 7 85 L 80.63 85" fill="none" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 85.88 85 L 78.88 88.5 L 80.63 85 L 78.88 81.5 Z" fill="rgb(0, 0, 0)" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="all"/><g transform="translate(-0.5 -0.5)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 1px; height: 1px; padding-top: 85px; margin-left: 47px;"><div data-drawio-colors="color: rgb(0, 0, 0); background-color: rgb(255, 255, 255); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 11px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; background-color: rgb(255, 255, 255); white-space: nowrap;">mqtt</div></div></div></foreignObject><text x="47" y="88" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="11px" text-anchor="middle">mqtt</text></switch></g><path d="M 121 165 L 121 200 L 161 200 L 161 171.37" fill="none" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 161 166.12 L 164.5 173.12 L 161 171.37 L 157.5 173.12 Z" fill="rgb(0, 0, 0)" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="all"/><rect x="117" y="70" width="160" height="30" rx="4.5" ry="4.5" fill="rgb(255, 255, 255)" stroke="rgb(0, 0, 0)" transform="rotate(-90,197,85)" pointer-events="all"/><g transform="translate(-0.5 -0.5)rotate(-90 197 85)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 158px; height: 1px; padding-top: 85px; margin-left: 118px;"><div data-drawio-colors="color: rgb(0, 0, 0); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; white-space: normal; overflow-wrap: normal;">facade</div></div></div></foreignObject><text x="197" y="89" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="12px" text-anchor="middle">facade</text></switch></g><rect x="293" y="70" width="160" height="30" rx="4.5" ry="4.5" fill="rgb(255, 255, 255)" stroke="rgb(0, 0, 0)" transform="rotate(-90,373,85)" pointer-events="all"/><g transform="translate(-0.5 -0.5)rotate(-90 373 85)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 158px; height: 1px; padding-top: 85px; margin-left: 294px;"><div data-drawio-colors="color: rgb(0, 0, 0); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; white-space: normal; overflow-wrap: normal;">device management</div></div></div></foreignObject><text x="373" y="89" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="12px" text-anchor="middle">device management</text></switch></g><rect x="331" y="70" width="160" height="30" rx="4.5" ry="4.5" fill="rgb(255, 255, 255)" stroke="rgb(0, 0, 0)" transform="rotate(-90,411,85)" pointer-events="all"/><g transform="translate(-0.5 -0.5)rotate(-90 411 85)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 158px; height: 1px; padding-top: 85px; margin-left: 332px;"><div data-drawio-colors="color: rgb(0, 0, 0); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; white-space: normal; overflow-wrap: normal;">decoder</div></div></div></foreignObject><text x="411" y="89" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="12px" text-anchor="middle">decoder</text></switch></g><rect x="368" y="70" width="160" height="30" rx="4.5" ry="4.5" fill="rgb(255, 255, 255)" stroke="rgb(0, 0, 0)" transform="rotate(-90,448,85)" pointer-events="all"/><g transform="translate(-0.5 -0.5)rotate(-90 448 85)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 158px; height: 1px; padding-top: 85px; margin-left: 369px;"><div data-drawio-colors="color: rgb(0, 0, 0); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; white-space: normal; overflow-wrap: normal;">converter</div></div></div></foreignObject><text x="448" y="89" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="12px" text-anchor="middle">converter</text></switch></g><path d="M 358 45 L 218.37 45" fill="none" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 213.12 45 L 220.12 41.5 L 218.37 45 L 220.12 48.5 Z" fill="rgb(0, 0, 0)" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="all"/><g transform="translate(-0.5 -0.5)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 1px; height: 1px; padding-top: 45px; margin-left: 285px;"><div data-drawio-colors="color: rgb(0, 0, 0); background-color: rgb(255, 255, 255); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 11px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; background-color: rgb(255, 255, 255); white-space: nowrap;">GET device config</div></div></div></foreignObject><text x="285" y="48" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="11px" text-anchor="middle">GET device...</text></switch></g><path d="M 212 85 L 351.63 85" fill="none" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 356.88 85 L 349.88 88.5 L 351.63 85 L 349.88 81.5 Z" fill="rgb(0, 0, 0)" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="all"/><g transform="translate(-0.5 -0.5)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 1px; height: 1px; padding-top: 85px; margin-left: 285px;"><div data-drawio-colors="color: rgb(0, 0, 0); background-color: rgb(255, 255, 255); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 11px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; background-color: rgb(255, 255, 255); white-space: nowrap;">PUB status</div></div></div></foreignObject><text x="285" y="88" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="11px" text-anchor="middle">PUB status</text></switch></g><rect x="87" y="0" width="390" height="170" rx="25.5" ry="25.5" fill="none" stroke="rgb(0, 0, 0)" pointer-events="all"/><path d="M 477 85 L 514.63 85" fill="none" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="stroke"/><path d="M 519.88 85 L 512.88 88.5 L 514.63 85 L 512.88 81.5 Z" fill="rgb(0, 0, 0)" stroke="rgb(0, 0, 0)" stroke-miterlimit="10" pointer-events="all"/><rect x="521" y="0" width="90" height="170" rx="13.5" ry="13.5" fill="none" stroke="rgb(0, 0, 0)" pointer-events="all"/><g transform="translate(-0.5 -0.5)"><switch><foreignObject pointer-events="none" width="100%" height="100%" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility" style="overflow: visible; text-align: left;"><div xmlns="http://www.w3.org/1999/xhtml" style="display: flex; align-items: unsafe center; justify-content: unsafe center; width: 88px; height: 1px; padding-top: 85px; margin-left: 522px;"><div data-drawio-colors="color: rgb(0, 0, 0); " style="box-sizing: border-box; font-size: 0px; text-align: center;"><div style="display: inline-block; font-size: 12px; font-family: Helvetica; color: rgb(0, 0, 0); line-height: 1.2; pointer-events: all; white-space: normal; overflow-wrap: normal;">iot-core</div></div></div></foreignObject><text x="566" y="89" fill="rgb(0, 0, 0)" font-family="Helvetica" font-size="12px" text-anchor="middle">iot-core</text></switch></g></g><switch><g requiredFeatures="http://www.w3.org/TR/SVG11/feature#Extensibility"/><a transform="translate(0,-5)" xlink:href="https://www.diagrams.net/doc/faq/svg-export-text-problems" target="_blank"><text text-anchor="middle" font-size="10px" x="50%" y="100%">Text is not SVG - cannot display</text></a></switch></svg>
## Dependencies  
 - [iot-device-mgmt](https://github.com/diwise/iot-device-mgmt)
 - [RabbitMQ](https://www.rabbitmq.com/)
# Facades
Since application servers such as [Chirpstack](https://www.chirpstack.io/application-server/) has different uplink payloads a facade is used to transform the specific payload into an internal format.
## Chirpstack
Support for Chirpstack v3 payloads.
## Netmore
Support for payloads from [netmore](https://netmoregroup.com/iot-network/)
# Codecs
Codec implementations for sensors
## Presence
[]() Uses codec for Sensative
## Qalcosonic
Decoder for Ambiductor Qalcosonic w1 water meters (w1e, w1t, w1h).
 - Volume (incl. timestamp)
 - Temperature (w1t)
 - Status (codes & messages)
## Elsys
 - Temperature         
 - ExternalTemperature 
 - Vdd                 
 - CO2                 
 - Humidity            
 - Light               
 - Motion              
 - Occupancy           
 - DigitalInput        
 - DigitalInputCounter 

Depends on the [Generic Javascript decoder](https://www.elsys.se/en/elsys-payload/)
## Enviot
 - Battery     
 - Humidity    
 - SensorStatus
 - SnowHeight  
 - Temperature 
## Milesight
 - Temperature
 - Humidity   
 - CO2        
 - Battery    
## Senlab
 - Temperature
## Sensative
 - BatteryLevel
 - Temperature
 - Humidity
 - DoorReport
 - DoorAlarm
 - Presence
## Sensefarm
- BatteryVoltage 
- Resistances    
- SoilMoistures  
- Temperature    
# Converters
Converters converts sensor data to lwm2m measurements.
## AirQuality   
[urn:oma:lwm2m:ext:3428](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3428.xml)
## Conductivity 
[urn:oma:lwm2m:ext:3327](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3327.xml)
## DigitalInput 
[urn:oma:lwm2m:ext:3200](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3200.xml)
## Humidity     
[urn:oma:lwm2m:ext:3304](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3304.xml)
## Illuminance  
[urn:oma:lwm2m:ext:3301](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3301.xml)
## PeopleCount  
[urn:oma:lwm2m:ext:3434](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3434.xml)
## Presence     
[urn:oma:lwm2m:ext:3302](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3302.xml)
## Pressure     
[urn:oma:lwm2m:ext:3323](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3323.xml)
## Temperature  
[urn:oma:lwm2m:ext:3303](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3303.xml)
## Watermeter   
[urn:oma:lwm2m:ext:3424](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3424.xml)
# Build, compose and test
## Build
```bash
docker build -f deployments/Dockerfile . -t diwise/iot-agent:latest
```
## Compose
```bash
docker compose -f deployments/docker-compose.yaml up
```
## Test
```bash
curl -X 
```

# Configuration
## Environment variables
```json
"MQTT_DISABLED": "false", # enable/disable mqtt input 
"MQTT_HOST": "<broker hostname>",
"MQTT_PORT": "<broker port number>",
"MQTT_USER": "<username>",
"MQTT_PASSWORD": "<password>",
"MQTT_TOPIC_0": "topic-01/#", # configure mqtt topic names
"MQTT_TOPIC_1": "topic-02/#", # it possible to specify multiple topics
...
"MQTT_TOPIC_n": "topic-n/#",
"RABBITMQ_HOST": "<rabbit mq hostname>"
"DEV_MGMT_URL": "http://iot-device-mgmt:8080", 
"SERVICE_PORT": "<custom service port, default 8080>",
"MSG_FWD_ENDPOINT" : "http://iot-agent:8080/api/v0/messages",
"OAUTH2_TOKEN_URL": "http://keycloak:8080/realms/diwise-local/protocol/openid-connect/token",
"OAUTH2_CLIENT_ID": "diwise-devmgmt-api",
"OAUTH2_CLIENT_SECRET": "<client secret>",
"APPSERVER_FACADE": "<facade>" # configure application server, chripstack (default) or netmore
```
## CLI flags
none
## Files
none
# Links
[iot-agent](https://diwise.github.io/) on diwise.github.io


