#!/bin/sh
envsubst < /rotor_template.json | cat > /cluster_template.json
exec /usr/bin/python3 -u /sbin/my_init -- /usr/local/bin/rotor.sh
