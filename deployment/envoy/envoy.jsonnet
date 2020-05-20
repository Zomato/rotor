{
    "node": {
        "id": "envoy",
        "cluster": "default-cluster",
        "locality": {
            "zone": "default-zone"
        }
    },
    "admin": {
        "access_log_path": "/dev/null",
        "address": {
            "socket_address": {
                "address": "0.0.0.0",
                "port_value": 9901,
                "protocol": "TCP"
            }
        }
    },
    "static_resources": {
        "clusters": [
            {
                "name": std.extVar("ENVOY_ROTOR_CLUSTER"),
                "type": "LOGICAL_DNS",
                "connect_timeout": {
                    "seconds": 30
                },
                "lb_policy": "ROUND_ROBIN",
                "hosts": [
                    {
                        "socket_address": {
                            "protocol": "TCP",
                            "address": std.extVar("ENVOY_ROTOR_HOST"),
                            "port_value": std.parseInt(std.extVar("ENVOY_ROTOR_PORT")),
                        }
                    }
                ],
                "http2_protocol_options": {
                    "max_concurrent_streams": 10
                },
                "upstream_connection_options": {
                    "tcp_keepalive": {
                        "keepalive_probes": 1,
                        "keepalive_time": 10,
                        "keepalive_interval": 10
                    }
                }
            },
        ],
        "listeners": [
            {
                "name": "default-cluster:80",
                "address": {
                    "socket_address": {
                        "address": "0.0.0.0",
                        "port_value": 80
                    }
                },
                "filterChains": [
                    {
                        "filterChainMatch": {},
                        "filters": [
                            {
                                "name": "envoy.http_connection_manager",
                                "config": {
                                    "http_filters": [
                                        {
                                            "name": "envoy.cors",
                                            "config": {}
                                        },
                                        {
                                            "name": "envoy.router",
                                            "config": {}
                                        }
                                    ],
                                    "rds": {
                                        "config_source": {
                                            "api_config_source": {
                                                "grpc_services": [
                                                    {
                                                        "envoy_grpc": {
                                                            "cluster_name": std.extVar("ENVOY_ROTOR_CLUSTER")
                                                        }
                                                    }
                                                ],
                                                "refresh_delay": "30.000s",
                                                "api_type": "GRPC"
                                            }
                                        },
                                        "route_config_name": "default-cluster:80"
                                    },
                                    "stat_prefix": "egress"
                                }
                            }
                        ]
                    }
                ]
            },
        ]
    },
    "dynamic_resources": {
        "lds_config": {
            "api_config_source": {
                "api_type": "GRPC",
                "grpc_services": [
                    {
                        "envoy_grpc": {
                            "cluster_name": std.extVar("ENVOY_ROTOR_CLUSTER")
                        }
                    }
                ],
                "refresh_delay": {
                    "seconds": 10
                }
            }
        },
        "cds_config": {
            "api_config_source": {
                "api_type": "GRPC",
                "grpc_services": [
                    {
                        "envoy_grpc": {
                            "cluster_name": std.extVar("ENVOY_ROTOR_CLUSTER")
                        }
                    }
                ],
                "refresh_delay": {
                    "seconds": 10
                }
            }
        }
    }
}
