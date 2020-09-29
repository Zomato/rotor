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

// Package aws provides integrations with Amazon EC2 and ECS. See
// "rotor help aws" and "rotor help ecs" for usage.
package aws

import (
	"fmt"
	"github.com/turbinelabs/cli/command"
	tbnflag "github.com/turbinelabs/nonstdlib/flag"
	"github.com/turbinelabs/nonstdlib/flag/usage"
	tbnstrings "github.com/turbinelabs/nonstdlib/strings"
	"github.com/turbinelabs/rotor"
	"github.com/turbinelabs/rotor/updater"
)

const (
	defaultClusterTagNamespace = "tbn:cluster"
	delimiter                  = ":"

	awsDescription = `Connects to the AWS API in a given region and
updates Clusters stored in the Turbine Labs API at startup and periodically
thereafter.

EC2 instance tags are used to determine to which clusters an instance belongs.
An EC2 instance may belong to multiple clusters, serving traffic on multiple
ports. Cluster membership on a port is declared with a tag, of the form:

    "<namespace>:<cluster-name>:<port>"=""

The port must be numeric, and the cluster name cannot contain the delimiter.
The delimiter is ":" and the default namespace is "` + defaultClusterTagNamespace + `".

Tags of the following form will be added to the Instance in the appropriate
Cluster, as "<key>"="<value>":

    "<namespace>:<cluster-name>:<port>:<key>"="<value>"

If key/value tags are included, the cluster membership tag is optional.

Tags without the namespaced cluster/port prefix will be added to all Instances
in all Clusters to which the EC2 Instance belongs.

By default, all EC2 Instances in the VPC are examined, but additional filters
can be specified (see -filters).

Additionally, by default if AWS credentials are not passed via cli then the 
AWS's Go SDK will fall back to its default credential chain. This first pulls
from the environment then falls back to the task role and finally the instance
profile role.`
)

func AWSCmd(updaterFlags rotor.UpdaterFromFlags) *command.Cmd {
	runner := &awsRunner{
		config: &EC2Config{},
	}

	cmd := &command.Cmd{
		Name:        "aws",
		Summary:     "aws collector",
		Usage:       "[OPTIONS]",
		Description: awsDescription,
		Runner:      runner,
	}

	flags := tbnflag.Wrap(&cmd.Flags)
	flags.StringVar(
		&runner.config.namespace,
		"cluster-tag-namespace",
		defaultClusterTagNamespace,
		"The namespace for cluster tags",
	)

	flags.StringVar(
		&runner.config.vpcID,
		"vpc-id",
		"",
		usage.Required("The ID of the VPC in which rotor is running"),
	)

	flags.Var(
		&runner.config.filterStrs,
		"filters",
		"A comma-delimited list of key/value pairs, used to specify additional "+
			"EC2 Instances filters. Of the form `\"<key>=<value>,...\"`. "+
			"See http://goo.gl/kSCOHS for a discussion of available filters.",
	)

	flags.StringVar(
		&runner.config.awsRegion,
		"aws.region",
		"",
		usage.Required("The AWS region in which the binary is running"),
	)

	flags.StringVar(
		&runner.config.awsSecretAccessKey,
		"aws.secret-access-key",
		"",
		usage.Sensitive("The AWS API secret access key"),
	)

	flags.StringVar(
		&runner.config.awsAccessKeyId,
		"aws.access-key-id",
		"",
		usage.Sensitive("The AWS API access key ID"),
	)

	flags.StringVar(
		&runner.config.awsIAMRoleToAssume,
		"aws.iam-role-to-assume",
		"",
		usage.Sensitive("The AWS IAM Role to assume"),
	)

	runner.updaterFlags = updaterFlags

	return cmd
}

type awsRunner struct {
	updaterFlags rotor.UpdaterFromFlags
	config *EC2Config
}

type EC2Config struct {
	namespace string
	delimiter string
	vpcID string
	filterStrs tbnflag.Strings  // todo: implement the tbnflag interface to directly parse
	filters map[string][]string
	awsRegion          string
	awsSecretAccessKey string
	awsAccessKeyId     string
	awsIAMRoleToAssume string
}



func (r *awsRunner) Run(cmd *command.Cmd, args []string) command.CmdErr {
	filters, err := r.processFilters(r.config.filterStrs.Strings)
	if err != nil {
		return cmd.BadInput(err)
	}

	config := EC2ClustersProviderConfig{
		Namespace: r.config.namespace,
		Delimiter: delimiter,
		VpcID:     r.config.vpcID,
		Filters:   filters,
		Aws:       EC2AWSConfig{
			Region:          r.config.awsRegion,
			AccessKeyId:     r.config.awsAccessKeyId,
			SecretAccessKey: r.config.awsSecretAccessKey,
			IAMRoleToAssume: r.config.awsIAMRoleToAssume,
		},
	}

	if err := r.updaterFlags.Validate(); err != nil {
		return cmd.BadInput(err)
	}

	u, err := r.updaterFlags.Make()
	if err != nil {
		return cmd.Error(err)
	}

	c, err := NewEC2ClusterProvider(config)
	if err != nil {
		return cmd.Error(err)
	}

	updater.Loop(u, c.GetClusters)

	return command.NoError()
}

func (r *awsRunner) processFilters(strs []string) (map[string][]string, error) {
	filters := map[string][]string{}
	for _, str := range strs {
		key, value := tbnstrings.SplitFirstEqual(str)
		if key == "" || value == "" {
			return nil, fmt.Errorf("malformed filter: %q", str)
		}
		filters[key] = append(filters[key], value)
	}

	return filters, nil
}
