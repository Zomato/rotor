package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/turbinelabs/nonstdlib/log/console"
	"github.com/turbinelabs/rotor/pkg/cluster_provider"
	"sort"
	"strconv"
	"strings"
)
import "github.com/turbinelabs/api"

type EC2AWSConfig struct {
	Region string
	AccessKeyId string
	SecretAccessKey string
	IAMRoleToAssume string
}


type EC2ClustersProviderConfig struct {
	Namespace   string
	Delimiter string
	VpcID string
	Filters map[string][]string
	Aws        EC2AWSConfig
}

type ec2ClustersProvider struct {
	config EC2ClustersProviderConfig
	ec2Svc   awsEC2Client
}


func NewEC2ClusterProvider(config EC2ClustersProviderConfig) (cluster_provider.ClusterProvider, error) {
	provider := &ec2ClustersProvider{
		config:    config,
		ec2Svc: newAWSEC2ClientFromConfig(&config.Aws),
	}

	return provider, nil
}

func (e *ec2ClustersProvider) String() string {
	return fmt.Sprintf(
		"EC2ClustersProvider{namespace=%s, delimiter=%s, vpcID=%s, filters=%v}",
		e.config.Namespace,
		e.config.Delimiter,
		e.config.VpcID,
		e.config.Filters,
	)
}

func (e *ec2ClustersProvider) GetClusters() ([]api.Cluster, error) {
	params := &ec2.DescribeInstancesInput{Filters: e.mkFilters()}
	resp, err := e.ec2Svc.DescribeInstances(params)
	if err != nil {
		return nil, fmt.Errorf("error executing aws api list: %s", err.Error())
	}
	return e.reservationsToClusters(resp.Reservations), nil
}

func (e *ec2ClustersProvider) mkFilters() []*ec2.Filter {
	filters := []*ec2.Filter{
		// return only running instances
		{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running")},
		},
		// in the provided VPC
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(e.config.VpcID)},
		},
	}

	// add custom filters
	for key, values := range e.config.Filters {
		valuePtrs := []*string{}
		for _, value := range values {
			valuePtrs = append(valuePtrs, aws.String(value))
		}
		filters = append(filters, &ec2.Filter{Name: aws.String(key), Values: valuePtrs})
	}

	return filters
}

func (e *ec2ClustersProvider) reservationsToClusters(reservs []*ec2.Reservation) api.Clusters {
	clustersMap := map[string]*api.Cluster{}
	for _, res := range reservs {
		for _, inst := range res.Instances {
			e.processEC2Instance(clustersMap, inst)
		}
	}

	clusters := make(api.Clusters, 0, len(clustersMap))
	for _, cluster := range clustersMap {
		sort.Sort(api.InstancesByHostPort(cluster.Instances))
		clusters = append(clusters, *cluster)
	}
	sort.Sort(api.ClusterByName(clusters))

	return clusters
}

func (e *ec2ClustersProvider) processEC2Instance(clusters map[string]*api.Cluster, inst *ec2.Instance) {
	host := *inst.PrivateIpAddress
	tpm := tagAndPortMap{prefix: e.config.Namespace, delimiter: e.config.Delimiter}

	// process all tags, extracting cluster-namespaced key/value pairs and ports
	for _, tag := range inst.Tags {
		// TODO: consider adding other machine metadata as tags
		if err := tpm.processTag(*tag.Key, *tag.Value); err != nil {
			console.Error().Printf("Skipping tag for Instance %s: %s", host, err)
		}
	}

	for clusterAndPort, md := range tpm.clusterTagMap {
		metadata := api.MetadataFromMap(md)
		for key, value := range tpm.globalTagMap {
			metadata = append(metadata, api.Metadatum{Key: key, Value: value})
		}
		sort.Sort(api.MetadataByKey(metadata))

		instance := api.Instance{
			Host:     host,
			Port:     clusterAndPort.port,
			Metadata: metadata,
		}

		clusterName := clusterAndPort.cluster
		cluster := clusters[clusterName]
		if cluster == nil {
			cluster = &api.Cluster{
				Name:      clusterName,
				Instances: []api.Instance{},
			}
			clusters[clusterName] = cluster
		}

		cluster.Instances = append(cluster.Instances, instance)
	}
}

type clusterAndPort struct {
	cluster string
	port    int
}

func newClusterAndPort(terms []string) (clusterAndPort, error) {
	nope := clusterAndPort{}

	if len(terms) < 2 {
		return nope, errors.New("must have at least cluster and port")
	}

	port, err := strconv.ParseUint(terms[1], 10, 16)
	if err != nil {
		return nope, fmt.Errorf("bad port: %s", err)
	}
	if port == 0 {
		return nope, fmt.Errorf("port must be non zero")
	}

	if terms[0] == "" {
		return nope, errors.New("cluster must be non-empty")
	}

	return clusterAndPort{terms[0], int(port)}, nil
}

// encapsulates extracting tags and ports from prefixed keys and values for a
// single instance.
type tagAndPortMap struct {
	prefix        string
	delimiter     string
	clusterTagMap map[clusterAndPort]map[string]string
	globalTagMap  map[string]string
}

func (tpm *tagAndPortMap) processTag(key, value string) error {
	if key == "" {
		return fmt.Errorf("empty tag key for value: %q", value)
	}

	prefixWithDelim := tpm.prefix + tpm.delimiter

	// if it doesn't have the prefix, it's a global tag
	if !strings.HasPrefix(key, prefixWithDelim) {
		if tpm.globalTagMap == nil {
			tpm.globalTagMap = map[string]string{}
		}
		tpm.globalTagMap[key] = value
		return nil
	}

	// remove the prefix
	suffix := key[len(prefixWithDelim):]
	if suffix == "" {
		return fmt.Errorf("tag key empty after %q prefix removed: %q=%q", prefixWithDelim, key, value)
	}

	terms := strings.SplitN(suffix, tpm.delimiter, 3)
	if len(terms) < 2 {
		return fmt.Errorf("tag key must have at least cluster name and port: %q=%q", key, value)
	}

	candp, err := newClusterAndPort(terms)
	if err != nil {
		return fmt.Errorf("malformed cluster/port in tag key: %q: %s", suffix, err)
	}

	if tpm.clusterTagMap == nil {
		tpm.clusterTagMap = map[clusterAndPort]map[string]string{}
	}

	if tpm.clusterTagMap[candp] == nil {
		tpm.clusterTagMap[candp] = map[string]string{}
	}

	if len(terms) > 2 {
		k := terms[2]
		if k == "" {
			return fmt.Errorf("tag key cluster name and port, but empty key: %q", key)
		}

		tpm.clusterTagMap[candp][k] = value
	}

	return nil
}
