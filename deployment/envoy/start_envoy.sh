#!/bin/sh

jsonnet $(env | sed 's/=.*//' | sed 's/^/--ext-str /' | tr '\n' ' ') /etc/envoy/envoy.jsonnet -o /etc/envoy/envoy.json
exec /usr/local/bin/envoy -c /etc/envoy/envoy.json --restart-epoch $RESTART_EPOCH --drain-time-s 120 --parent-shutdown-time-s 130
