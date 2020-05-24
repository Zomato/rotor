package multi

import (
	"fmt"
	"github.com/turbinelabs/rotor/plugins/aws"
)

type OneOfClustersProviderConfig interface {
	GetClustersProviderConfigType() ConfigType
	GetEC2ClustersProviderConfig() *aws.EC2ClustersProviderConfig
	GetECSClustersProviderConfig() *aws.ECSClustersProviderConfig
	//GetKubernetesClustersProviderConfig()
	//GetEKSClustersProviderConfig()
	//GetConsulClustersProviderConfig()
}


type ConfigType string

const EC2ClustersProviderConfigType ConfigType = "EC2ClustersProvider"
const ECSClustersProviderConfigType ConfigType = "ECSClustersProvider"

func NewOneOfEC2ClustersProviderConfig(config *aws.EC2ClustersProviderConfig) OneOfClustersProviderConfig {
	return &oneOfImpl{
		_type:                     EC2ClustersProviderConfigType,
		ec2ClustersProviderConfig: config,
		ecsClustersProviderConfig: nil,
	}
}

func NewOneOfECSClustersProviderConfig(config *aws.ECSClustersProviderConfig) OneOfClustersProviderConfig {
	return &oneOfImpl{
		_type:                     EC2ClustersProviderConfigType,
		ec2ClustersProviderConfig: nil,
		ecsClustersProviderConfig: config,
	}
}

type oneOfImpl struct {
	_type                     ConfigType
	ec2ClustersProviderConfig *aws.EC2ClustersProviderConfig
	ecsClustersProviderConfig *aws.ECSClustersProviderConfig
}

func (o oneOfImpl) GetClustersProviderConfigType() ConfigType {
	return o._type
}

func (o oneOfImpl) GetEC2ClustersProviderConfig() *aws.EC2ClustersProviderConfig {
	if o._type != EC2ClustersProviderConfigType {
		panic(fmt.Sprintf(
			"OneOfClustersProviderConfig: wrong config requested for the config, expected: %s, actual: %s",
			EC2ClustersProviderConfigType, o._type,
		))
	}
	return o.ec2ClustersProviderConfig
}

func (o oneOfImpl) GetECSClustersProviderConfig() *aws.ECSClustersProviderConfig {
	if o._type != ECSClustersProviderConfigType {
		panic(fmt.Sprintf(
			"OneOfClustersProviderConfig: wrong config requested for the config, expected: %s, actual: %s",
			ECSClustersProviderConfigType, o._type,
		))
	}
	return o.ecsClustersProviderConfig
}
