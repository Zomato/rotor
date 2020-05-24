package aws

import (
	"fmt"
	"github.com/turbinelabs/api"
	"github.com/turbinelabs/nonstdlib/log/console"
	"github.com/turbinelabs/rotor/pkg/cluster_provider"
)

// todo remove struct duplication by migrating to independent packages
type ECSAWSConfig struct {
	Region string           `json:"region"`
	AccessKeyId string      `json:"access_key_id"`
	SecretAccessKey string  `json:"secret_access_key"`
	IAMRoleToAssume string  `json:"iam_role_to_assume"`
}

type ECSClustersProviderConfig struct {
	Clusters   []string      `json:"clusters"`
	ClusterTag string        `json:"cluster_tag"`
	Aws        ECSAWSConfig  `json:"aws"`
}


type ecsClusterProvider struct {
	config ECSClustersProviderConfig
	awsClient awsECSClient
}

func (e *ecsClusterProvider) String() string {
	return fmt.Sprintf(
		"ECSClusterProvider{clusters=%v, cluster_tag=%s, aws_region=%s, aws_access_key_id=%s, iam_role_to_assume=%s}",
		e.config.Clusters,
		e.config.ClusterTag,
		e.config.Aws.Region,
		e.config.Aws.AccessKeyId,
		e.config.Aws.IAMRoleToAssume,
	)
}

func NewECSClusterProvider(config ECSClustersProviderConfig) (cluster_provider.ClusterProvider, error) {
	provider := &ecsClusterProvider{
		config: config,
		awsClient: newAWSECSClientFromConfig(&config.Aws),
	}
	if err := provider.validateConfig(); err != nil {
		return nil, err
	}
	return provider, nil
}

func (e *ecsClusterProvider) validateConfig() error {
	clusters, err := e.awsClient.ListClusters()
	if err != nil {
		return err
	}

	for _, c := range e.config.Clusters {
		if _, match := clusters[c]; !match {
			return fmt.Errorf("ECS cluster %s was not found", c)
		}
	}

	return nil
}

func (e *ecsClusterProvider) GetClusters() ([]api.Cluster, error) {
	state, err := NewECSState(e.awsClient, e.config.Clusters)
	if err != nil {
		return nil, fmt.Errorf("Could not read ECS state: %v", err.Error())
	}

	// super fucking hack
	tagSet := state.meta.identifyTaggedItems(e.config.ClusterTag)
	for i := len(tagSet) - 1; i >= 0; i-- {
		clusterTemplate := tagSet[i]
		if err := state.validate(clusterTemplate); err != nil {
			console.Error().Println(err)
			tagSet = append(tagSet[:i], tagSet[i+1:]...)
		}
	}

	return bindClusters(e.config.ClusterTag, state, tagSet), nil
}

