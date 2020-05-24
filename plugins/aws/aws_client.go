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

//go:generate $TBN_HOME/scripts/mockgen_internal.sh -type client -source $GOFILE -destination mock_$GOFILE -package $GOPACKAGE --write_package_comment=false

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	ec2 "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	tbnflag "github.com/turbinelabs/nonstdlib/flag"
)

// client represents the command-line flags specifying configuration of
// an AWS client and its underlying session.
type client interface {
	// MakeAWSEC2Client produces an EC2 interface from a new AWS client session.
	MakeAWSEC2Client() awsEC2Client

	// MakeAWSECSClient produces an AWS interface from a new AWS client session.
	MakeAWSECSClient() awsECSClient
}

// newClientFromFlags produces a client, adding necessary flags to the
// provided flag.FlagSet.
func newClientFromFlags(fs tbnflag.FlagSet) client {
	return &clientImpl{}
}

func newAWSECSClientFromConfig(config *ECSAWSConfig) awsECSClient {
	ff := &clientImpl{}

	ff.awsRegion = config.Region
	ff.awsSecretAccessKey = config.SecretAccessKey
	ff.awsAccessKeyID = config.AccessKeyId
	ff.awsIamRoleToAssume = config.IAMRoleToAssume

	return ff.MakeAWSECSClient()
}

func newAWSEC2ClientFromConfig(config *EC2AWSConfig) awsEC2Client {
	ff := &clientImpl{}

	ff.awsRegion = config.Region
	ff.awsSecretAccessKey = config.SecretAccessKey
	ff.awsAccessKeyID = config.AccessKeyId
	ff.awsIamRoleToAssume = config.IAMRoleToAssume

	return ff.MakeAWSEC2Client()
}

type clientImpl struct {
	awsRegion          string
	awsSecretAccessKey string
	awsAccessKeyID     string
	awsIamRoleToAssume string
}

func (ff *clientImpl) makeSession() *session.Session {
	sessForSTSCreds := session.New(&aws.Config{
		Region:      aws.String(ff.awsRegion),
		Credentials: ff.awsCredentials(),
	})

	creds := stscreds.NewCredentials(sessForSTSCreds, ff.awsIamRoleToAssume)
	sess := session.New(&aws.Config{
		Credentials:                       creds,
		Region:                            aws.String(ff.awsRegion),
	})

	return sess
}

func (ff *clientImpl) awsCredentials() *credentials.Credentials {
	// This gets all the AWS Defaults. They will be merged correctly with
	// awsSession on the call to `session.New()
	defaultConfig := defaults.Config()
	defaultHandlers := defaults.Handlers()

	customProvider := &credentials.StaticProvider{
		Value: credentials.Value{
			AccessKeyID:     ff.awsAccessKeyID,
			SecretAccessKey: ff.awsSecretAccessKey,
		},
	}

	// Unfortunately AWS doesn't have a variable for its default providers.
	// So this mimics the latest provider chain in the defaults package
	// located at
	// https://github.com/aws/aws-sdk-go/blob/d856824058f17a35c61cabdfb1c40559ce070cd9/aws/defaults/defaults.go#L95-L99
	// This takes the default chain, and adds the `legacy` cli way to highest
	// hierachy. Currently have an issue to address this in aws-sdk-go
	// https://github.com/aws/aws-sdk-go/issues/2051
	return credentials.NewCredentials(
		&credentials.ChainProvider{
			VerboseErrors: true,
			Providers: []credentials.Provider{
				customProvider,
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
				defaults.RemoteCredProvider(*defaultConfig, defaultHandlers),
			},
		},
	)
}

func (ff *clientImpl) MakeAWSEC2Client() awsEC2Client {
	return ec2.New(ff.makeSession())
}

func (ff *clientImpl) MakeAWSECSClient() awsECSClient {
	s := ff.makeSession()
	return newAwsClient(ecs.New(s), ec2.New(s))
}
