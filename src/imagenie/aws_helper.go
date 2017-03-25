package main

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"net/http"
	"os"
	"path"
)

type AwsConfig struct {
	AccessKey string
	Secret    string
	Region    string
	Bucket    string
}

type AwsHelper struct {
	config AwsConfig
	svc    *s3.S3
}

func (self *AwsHelper) Init(config AwsConfig) error {
	log.Info("Initializing AWS client with credentials")
	self.config = config
	creds := credentials.NewStaticCredentials(self.config.AccessKey, self.config.Secret, "")

	_, err := creds.Get()
	if err != nil {
		log.Error("Aws credentials error: ", err)
		return err
	}

	cfg := aws.NewConfig().WithRegion(self.config.Region).WithCredentials(creds)
	self.svc = s3.New(session.New(), cfg)
	return nil
}

func (self *AwsHelper) UploadFile(file_name, folder string) error {
	file, err := os.Open(file_name)
	if err != nil {
		return err
	}
	defer file.Close()

	file_info, err := file.Stat()
	if err != nil {
		return err
	}

	size := file_info.Size()

	buffer := make([]byte, size)
	file.Read(buffer)

	file_bytes := bytes.NewReader(buffer)
	file_type := http.DetectContentType(buffer)
	file_path := path.Clean(folder + "/" + path.Base(file_name))

	params := &s3.PutObjectInput{
		Bucket:        aws.String(self.config.Bucket),
		Key:           aws.String(file_path),
		Body:          file_bytes,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(file_type),
	}

	_, err = self.svc.PutObject(params)
	if err != nil {
		return err
	}
	return nil
}

func (self *AwsHelper) DeleteFile(file_name, folder string) error {
	file_path := path.Clean(folder + "/" + path.Base(file_name))

	params := &s3.DeleteObjectInput{
		Bucket: aws.String(self.config.Bucket),
		Key:    aws.String(file_path),
	}

	// Add some more retries
	_, err := self.svc.DeleteObject(params)
	if err != nil {
		log.Error("Error deleting file from S3: ", file_path)
		return err
	}
	return nil
}

func (self *AwsHelper) CheckFile(file_name, folder string) bool {
	file_path := path.Clean(folder + "/" + path.Base(file_name))

	params := &s3.HeadObjectInput{
		Bucket: aws.String(self.config.Bucket),
		Key:    aws.String(file_path),
	}

	_, err := self.svc.HeadObject(params)
	if err != nil {
		return false
	}

	log.Info("File already exists on S3: ", file_path)
	return true
}
