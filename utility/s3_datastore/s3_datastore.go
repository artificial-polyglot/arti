package s3_datastore

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"

	"io"
	"os"
	"strings"

	log "github.com/artificial-polyglot/arti/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Client struct {
	ctx    context.Context
	Config aws.Config
	Client *s3.Client
}

/**
This class is compatible with any version of S3: AWS, R2, etc.
Notice that it expects environment variables, not a .aws config
Also, note the key and secret key environment variables are slightly
revised, so that they could co-exist with the AWS names

When using this with AWS S3, the s3_endpoint_url should be set to ""
*/

func NewS3Client(ctx context.Context) (S3Client, *log.Status) {
	var cl S3Client
	cl.ctx = ctx
	var err error
	endpoint := os.Getenv("S3_ENDPOINT_URL")
	region := os.Getenv("S3_REGION")
	accessKey := os.Getenv("S3_ACCESS_KEY_ID")
	secretKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	cl.Config, err = config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return cl, log.Error(ctx, 500, err, "NewS3Client error")
	}
	cl.Client = s3.NewFromConfig(cl.Config, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = true
	})
	return cl, nil
}

func (t S3Client) GetObject(bucket string, key string) ([]byte, *log.Status) {
	var result []byte
	response, err := t.Client.GetObject(t.ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return result, log.Error(t.ctx, 500, err, "Error Getting Object", key)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return result, log.Error(t.ctx, 500, err, "Error Reading Object", key)
	}
	return body, nil
}

func (t S3Client) ListObjects(bucket, prefix string) ([]types.Object, *log.Status) {
	var results []types.Object
	list, err := t.Client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		status := log.Error(t.ctx, 500, err, "Error listing S3 bucket objects.")
		return results, status
	}
	for _, obj := range list.Contents {
		//key := aws.ToString(obj.Key)
		results = append(results, obj)
	}
	return results, nil
}

func (t S3Client) ListPrefixes(bucket, prefix string) ([]string, *log.Status) {
	var results []string
	list, err := t.Client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(`/`),
	})
	if err != nil {
		return results, log.Error(t.ctx, 500, err, "Error Listing S3 object prefix", prefix)
	}
	for _, obj := range list.CommonPrefixes {
		pref := aws.ToString(obj.Prefix)
		results = append(results, pref)
	}
	return results, nil
}

func (t S3Client) PutString(bucket string, key string, content string) *log.Status {
	input := &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   strings.NewReader(content),
	}
	_, err := t.Client.PutObject(t.ctx, input)
	if err != nil {
		return log.Error(t.ctx, 500, err, "Error uploading content to S3 key:", key)
	}
	return nil
}

func (t S3Client) PutFile(bucket string, key string, filePath string) *log.Status {
	var status *log.Status
	file, err := os.Open(filePath)
	if err != nil {
		return log.Error(t.ctx, 500, err, "Error opening file: ", filePath, "to upload to S3.")
	}
	defer file.Close()
	_, err = t.Client.PutObject(t.ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   file,
	})
	if err != nil {
		status = log.Error(t.ctx, 500, err, "Error uploading file to S3 key:", key)
	}
	return status
}

func (t S3Client) PutDirectory(bucket, prefix, localDir string) *log.Status {
	var status *log.Status
	filepath.WalkDir(localDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			status = log.Error(t.ctx, 500, err, "UploadDirectory walk error")
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(localDir, path)
		key := prefix + "/" + filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			status = log.Error(t.ctx, 500, err, "UploadDirectory read file: "+path)
			return err
		}
		_, err = t.Client.PutObject(t.ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(data),
		})
		if err != nil {
			status = log.Error(t.ctx, 500, err, "UploadDirectory put object: "+key)
			return err
		}
		return nil
	})
	return status
}
