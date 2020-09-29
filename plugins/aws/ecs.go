/*
Copyright 2018 Turbine Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"github.com/turbinelabs/cli/command"
	tbnflag "github.com/turbinelabs/nonstdlib/flag"
	"github.com/turbinelabs/nonstdlib/flag/usage"
	"github.com/turbinelabs/rotor"
	"github.com/turbinelabs/rotor/updater"
)

const ecsDefaultClusterTag = "tbn-cluster"

type ecsRunner struct {
	updaterFlags rotor.UpdaterFromFlags
	ecsConfig    *ECSConfig
}

type ECSConfig struct {
	clusters tbnflag.Strings
	clusterTag         string
	clusterPortTag     string
		awsRegion          string
		awsSecretAccessKey string
		awsAccessKeyId     string
		awsIAMRoleToAssume string
}

func ECSCmd(updaterFlags rotor.UpdaterFromFlags) *command.Cmd {
	runner := &ecsRunner{}
	cmd := &command.Cmd{
		Name:        "ecs",
		Summary:     "ECS collector",
		Usage:       "[OPTIONS]",
		Description: ecsDescription,
		Runner:      runner,
	}
	runner.ecsConfig = &ECSConfig{}

	runner.ecsConfig.clusters = tbnflag.NewStrings()

	flags := tbnflag.Wrap(&cmd.Flags)

	flags.Var(
		&runner.ecsConfig.clusters,
		"clusters",
		usage.Required(
			"Specifies a comma separated list indicating which ECS clusters "+
				"should be examined for containers marked for inclusion as API clusters. "+
				"No value means all clusters will be examined.",
		),
	)

	flags.StringVar(
		&runner.ecsConfig.clusterTag,
		"cluster-tag",
		ecsDefaultClusterTag,
		"label indicating what API clusters an instance of this container will serve")

	flags.StringVar(
		&runner.ecsConfig.awsRegion,
		"aws.region",
		"",
		usage.Required("The AWS region in which the binary is running"),
	)

	flags.StringVar(
		&runner.ecsConfig.awsSecretAccessKey,
		"aws.secret-access-key",
		"",
		usage.Sensitive("The AWS API secret access key"),
	)

	flags.StringVar(
		&runner.ecsConfig.awsAccessKeyId,
		"aws.access-key-id",
		"",
		usage.Sensitive("The AWS API access key ID"),
	)

	flags.StringVar(
		&runner.ecsConfig.awsIAMRoleToAssume,
		"aws.iam-role-to-assume",
		"",
		usage.Sensitive("The AWS IAM Role to assume"),
	)

	runner.updaterFlags = updaterFlags

	return cmd
}

func (r ecsRunner) Run(cmd *command.Cmd, args []string) command.CmdErr {
	if err := r.updaterFlags.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	clustersProvider, err := NewECSClusterProvider(ECSClustersProviderConfig{
		Clusters:   r.ecsConfig.clusters.Strings,
		ClusterTag: r.ecsConfig.clusterTag,
		Aws:        ECSAWSConfig{
			Region: r.ecsConfig.awsRegion,
			AccessKeyId: r.ecsConfig.awsAccessKeyId,
			SecretAccessKey: r.ecsConfig.awsSecretAccessKey,
			IAMRoleToAssume: r.ecsConfig.awsIAMRoleToAssume,
		},
	})

	if err != nil {
		return cmd.BadInput(err)
	}

	u, err := r.updaterFlags.Make()
	if err != nil {
		return cmd.Error(err)
	}

	updater.Loop(u, clustersProvider.GetClusters)

	return command.NoError()
}
