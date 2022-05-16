package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

func NewAwsClient(regionId, accessKey, secretKey string) (ec2Client *ec2.Client, eksClient *eks.Client) {
	credential := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg := aws.Config{
		Region: regionId,
		Credentials: credential,
	}
	return ec2.NewFromConfig(cfg), eks.NewFromConfig(cfg)
}
