package resource

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/aws"
	"cloud-manager/app/modules/jumpserver"
	"cloud-manager/app/util/response"
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"path/filepath"
)

func S3JumpBackup(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	fmt.Println("---Start backup jump host ---")
	if len(config.G.AwsInfo) <= 0 {
		utilGin.Response(BadRequest, "not found aws auth", nil)
		return
	}
	auth := config.G.AwsInfo[0]
	if len(auth.S3) <= 0 {
		utilGin.Response(BadRequest, "not found s3 info", nil)
		return
	}

	jumpClient := jumpserver.NewJmsClient(config.G.JumpInfo, config.DEFAULT)
	client, err := aws.NewS3("cn-north-1", auth.AccessKeyId, auth.AccessKeySecret)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}

    file, err := jumpClient.ExportHosts("/tmp")
    if err != nil {
		utilGin.Response(RequestFailed, err.Error(), nil)
		return
	}
	bucket := auth.S3[0].Bucket
	object := auth.S3[0].Object
    //bucket := "data-backup-data"
    objectKey := fmt.Sprintf("%s/%s", object, filepath.Base(file))
    fmt.Printf("backup to %s...\n", objectKey)
	if fileIO, err := os.Open(file); err == nil {
		err = client.UploadFileInS3Bucket(fileIO, bucket, objectKey)
		if err != nil {
			fmt.Println("Upload S3 err:", err)
		} else {
			fmt.Println("Upload s3 success")
		}
		_ = fileIO.Close()
		_ = os.Remove(file)
	}
	utilGin.Response(Success, "", nil)
}
