package cluster_provider

import "github.com/turbinelabs/api"

type ClusterProvider interface {
	String() string
	GetClusters() ([]api.Cluster, error)
}
