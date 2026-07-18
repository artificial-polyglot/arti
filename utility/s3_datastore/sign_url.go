package s3_datastore

import (
	"path/filepath"
	"strings"
	"time"

	log "github.com/artificial-polyglot/arti/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (t S3Client) SignAudioURL(baseURL string, filename string) string {
	if strings.HasPrefix(baseURL, "file://") {
		baseURL = baseURL[7:]
		return filepath.Join(baseURL, filename)
	}
	if strings.HasPrefix(baseURL, "s3://") {
		baseURL = baseURL[5:]
		slash := strings.Index(baseURL, "/")
		bucket := baseURL[0:slash]
		prefix := baseURL[slash+1:]
		var objectKey string
		if strings.HasSuffix(prefix, "/") {
			objectKey = prefix + filename
		} else {
			objectKey = prefix + "/" + filename
		}
		return t.SignS3AudioURL(bucket, objectKey)
	}
	return filepath.Join(baseURL, filename)
}

func (t S3Client) SignS3AudioURL(bucket string, objectKey string) string {
	ps := s3.NewPresignClient(t.Client)
	signed, err := ps.PresignGetObject(t.ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(bucket),
		Key:                        aws.String(objectKey),
		ResponseContentType:        aws.String("audio/mpeg"),
		ResponseContentDisposition: aws.String("inline"),
	}, s3.WithPresignExpires(time.Hour*24*7))
	if err != nil {
		log.Warn(t.ctx, "Unable to create presigned URL for", bucket, objectKey)
	}
	return signed.URL
}
