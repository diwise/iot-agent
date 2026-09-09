// Package lwm2m is the stable public SenML/LwM2M object model for
// iot-agent.
//
// It is consumed outside this module, notably by iot-core (function
// transforms and messaging contracts). Constructor signatures,
// object types, URN constants, package path and serialization format
// must not change without a compatibility/migration plan and
// verification against each consumer.
//
// Moving, renaming or deleting anything in this package requires a
// compatible release sequence: release iot-agent first, then upgrade
// and verify each consumer before removing old API.
package lwm2m
