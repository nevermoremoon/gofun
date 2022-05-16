package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
)

type S3 struct {
	Client *s3.Client
}

func NewS3(regionId, accessKey, secretKey string) (*S3, error) {
	credential := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg := aws.Config{
		Region: regionId,
		Credentials: credential,
	}
	client := s3.NewFromConfig(cfg)
	return &S3{Client: client}, nil
}

func (s *S3) UploadFileInS3Bucket(file io.Reader, bucket, objectKey string) (err error) {
	//fileName := "test123.jpg"
	//filePath := "/BUCKET_NAME/uploads/2021/6/25/"

	client := s.Client
	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(objectKey),
		Body:        file,
		//ContentType: aws.String("image"),
	})
	return err
}

