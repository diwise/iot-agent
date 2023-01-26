# iot-agent
A service that handles (decodes and converts) incoming data from multiple sources.

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://github.com/diwise/iot-agent/blob/main/LICENSE)

# Design
![svg](https://github.com/diwise/diwise.github.io/blob/main/site/static/images/iot-agent.svg)
## Dependencies  
 - [iot-device-mgmt](https://github.com/diwise/iot-device-mgmt)
 - [RabbitMQ](https://www.rabbitmq.com/)
# Facades
Since application servers such as [Chirpstack](https://www.chirpstack.io/application-server/) has different uplink payloads a facade is used to transform the specific payload into an internal format.
### Chirpstack
Support for Chirpstack v3 payloads.
### Netmore
Support for payloads from [netmore](https://netmoregroup.com/iot-network/)
# Codecs
Codec implementations for sensors
### Presence
[]() Uses codec for Sensative
### Qalcosonic
Decoder for Ambiductor Qalcosonic w1 water meters (w1e, w1t, w1h).
 - Volume (incl. timestamp)
 - Temperature (w1t)
 - Status (codes & messages)
### Elsys
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
### Enviot
 - Battery     
 - Humidity    
 - SensorStatus
 - SnowHeight  
 - Temperature 
### Milesight
 - Temperature
 - Humidity   
 - CO2        
 - Battery    
### Senlab
 - Temperature
### Sensative
 - BatteryLevel
 - Temperature
 - Humidity
 - DoorReport
 - DoorAlarm
 - Presence
### Sensefarm
- BatteryVoltage 
- Resistances    
- SoilMoistures  
- Temperature    
# Converters
Converters converts sensor data to lwm2m measurements.
### AirQuality   
[urn:oma:lwm2m:ext:3428](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3428.xml)
### Conductivity 
[urn:oma:lwm2m:ext:3327](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3327.xml)
### DigitalInput 
[urn:oma:lwm2m:ext:3200](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3200.xml)
### Humidity     
[urn:oma:lwm2m:ext:3304](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3304.xml)
### Illuminance  
[urn:oma:lwm2m:ext:3301](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3301.xml)
### PeopleCount  
[urn:oma:lwm2m:ext:3434](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3434.xml)
### Presence     
[urn:oma:lwm2m:ext:3302](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3302.xml)
### Pressure     
[urn:oma:lwm2m:ext:3323](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3323.xml)
### Temperature  
[urn:oma:lwm2m:ext:3303](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3303.xml)
### Watermeter   
[urn:oma:lwm2m:ext:3424](https://github.com/OpenMobileAlliance/lwm2m-registry/blob/prod/3424.xml)
# Build and test
## Build
```bash
docker build -f deployments/Dockerfile . -t diwise/iot-agent:latest
```
## Test
```bash
curl -X POST http://localhost:8080 
     -H "Content-Type: application/json"
     -d '{
            "deviceName": "mcg-ers-co2-01",
            "deviceProfileName": "ELSYS",
            "deviceProfileID": "0b765672",
            "devEUI": "a1b2c3d4e5f6",
            "data": "AQDuAhYEALIFAgYBxAcONA==",
            "object": {
                "co2": 452,
                "humidity": 22,
                "light": 178,
                "motion": 2,
                "temperature": 23.8,
                "vdd": 3636
            }
        }'
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
"RABBITMQ_HOST": "<rabbit mq hostname>",
"RABBITMQ_PORT": "5672",
"RABBITMQ_VHOST": "/",
"RABBITMQ_USER": "user",
"RABBITMQ_PASS": "bitnami",
"RABBITMQ_DISABLED": "false",
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
## Configuration files
none
# Links
[iot-agent](https://diwise.github.io/) on diwise.github.io


