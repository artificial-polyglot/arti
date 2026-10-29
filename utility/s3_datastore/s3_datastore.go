package s3_datastore

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io/fs"
	"path/filepath"

	"io"
	"os"
	"strconv"
	"strings"
	"time"

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

// HasModel reports whether a model file over 1Meg exists in the bucket under
// prefix, e.g. to check whether a trained model (such as mms_adapters/<lang>)
// has already been uploaded, regardless of whether it sits directly under
// the prefix or under a run-numbered subdirectory (see DownloadLatestFileTree).
func (t S3Client) HasModel(bucket string, prefix string) (bool, *log.Status) {
	trimmedPrefix := strings.TrimSuffix(prefix, "/") + "/"
	list, err := t.Client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(trimmedPrefix),
	})
	if err != nil {
		return false, log.Error(t.ctx, 500, err, "Error checking for existing model", prefix)
	}
	for _, obj := range list.Contents {
		if obj.Size != nil && *obj.Size > 1000000 { // must be GT 1Meg
			return true, nil
		}
	}
	return false, nil
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
	mimeType := "text/plain"
	disposition := "inline"
	input := &s3.PutObjectInput{
		Bucket:             &bucket,
		Key:                &key,
		Body:               strings.NewReader(content),
		ContentType:        &mimeType,
		ContentDisposition: &disposition,
	}
	_, err := t.Client.PutObject(t.ctx, input)
	if err != nil {
		return log.Error(t.ctx, 500, err, "Error uploading content to S3 key:", key)
	}
	return nil
}

func (t S3Client) PutFile(bucket string, key string, filePath string, mimeType string, inline bool) *log.Status {
	var status *log.Status
	file, err := os.Open(filePath)
	if err != nil {
		return log.Error(t.ctx, 500, err, "Error opening file: ", filePath, "to upload to S3.")
	}
	defer file.Close()
	if mimeType != "" {
		var disposition string
		if inline {
			disposition = "inline"
		} else {
			disposition = "attachment"
		}
		_, err = t.Client.PutObject(t.ctx, &s3.PutObjectInput{
			Bucket:             &bucket,
			Key:                &key,
			Body:               file,
			ContentType:        &mimeType,
			ContentDisposition: &disposition,
		})
	} else {
		_, err = t.Client.PutObject(t.ctx, &s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &key,
			Body:   file,
		})
	}
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
		var mimeType string
		var disposition string
		fileExt := strings.ToLower(filepath.Ext(path))
		if fileExt == ".sfm" || fileExt == ".usx" {
			mimeType = "text/plain"
			disposition = "inline"
		} else if fileExt == ".mp3" {
			mimeType = "audio/mpeg"
			disposition = "inline"
		} else if fileExt == ".wav" {
			mimeType = "audio/wav"
			disposition = "inline"
		} else if fileExt == ".json" {
			mimeType = "application/json"
			disposition = "inline"
		} else if fileExt == ".yaml" {
			mimeType = "application/yaml"
			disposition = "inline"
		} else {
			mimeType = "application/octet-stream"
			disposition = "attachment"
		}
		_, err = t.Client.PutObject(t.ctx, &s3.PutObjectInput{
			Bucket:             aws.String(bucket),
			Key:                aws.String(key),
			Body:               bytes.NewReader(data),
			ContentType:        aws.String(mimeType),
			ContentDisposition: aws.String(disposition),
		})
		if err != nil {
			status = log.Error(t.ctx, 500, err, "UploadDirectory put object: "+key)
			return err
		}
		return nil
	})
	return status
}

func (t S3Client) ClearDirectory(bucket string, prefix string) *log.Status {
	list, err := t.Client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return log.Error(t.ctx, 500, err, "Error listing S3 objects to clear", prefix)
	}
	if len(list.Contents) == 0 {
		return nil
	}
	var objects []types.ObjectIdentifier
	for _, obj := range list.Contents {
		objects = append(objects, types.ObjectIdentifier{Key: obj.Key})
	}
	_, err = t.Client.DeleteObjects(t.ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{Objects: objects},
	})
	if err != nil {
		return log.Error(t.ctx, 500, err, "Error deleting S3 objects under", prefix)
	}
	return nil
}

func (t S3Client) DownloadFileTree(bucket string, prefix string, localDir string) *log.Status {
	list, err := t.Client.ListObjectsV2(t.ctx, &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})
	if err != nil {
		return log.Error(t.ctx, 500, err, "Could not download", bucket, prefix)
	}
	if list == nil || len(list.Contents) == 0 {
		return log.ErrorNoErr(t.ctx, 500, err, "There was nothing to download", bucket, prefix)
	}
	for _, obj := range list.Contents {
		key := *obj.Key
		if strings.HasSuffix(key, "/") { // skip "folder" placeholder keys
			continue
		}
		rel := strings.TrimPrefix(key, prefix)
		localPath := filepath.Join(localDir, rel)
		err1 := t.DownloadFile(bucket, key, localPath)
		if err1 != nil {
			return err1
		}
	}
	return nil
}

// DownloadLatestFileTree is like DownloadFileTree, except the directory one
// level below prefix may be a 5-digit zero-filled run/version number (e.g.
// mms_adapters/atg/00005/). When such run directories exist, it selects the
// highest numbered one and downloads its contents, dropping the run number
// from the local path. When no run directories exist, it behaves exactly
// like DownloadFileTree.
func (t S3Client) DownloadLatestFileTree(bucket string, prefix string, localDir string) *log.Status {
	trimmedPrefix := strings.TrimSuffix(prefix, "/")
	prefixes, status := t.ListPrefixes(bucket, trimmedPrefix+"/")
	if status != nil {
		return status
	}
	maxRun := -1
	var maxRunStr string
	for _, p := range prefixes {
		parts := strings.Split(strings.TrimSuffix(p, "/"), "/")
		runStr := parts[len(parts)-1]
		if !isRunNumber(runStr) {
			continue
		}
		runNum, err := strconv.Atoi(runStr)
		if err != nil {
			return log.Error(t.ctx, 500, err, "Error converting run number to int; value:", runStr)
		}
		if runNum > maxRun {
			maxRun = runNum
			maxRunStr = runStr
		}
	}
	if maxRun < 0 {
		return t.DownloadFileTree(bucket, trimmedPrefix, localDir)
	}
	return t.DownloadFileTree(bucket, trimmedPrefix+"/"+maxRunStr, localDir)
}

func isRunNumber(s string) bool {
	if len(s) != 5 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

const (
	downloadChunkSize    = 64 * 1024 * 1024 // bytes per ranged GET
	downloadChunkTimeout = 5 * time.Minute  // a stalled chunk is abandoned and resumed after this
	downloadMaxAttempts  = 5                // consecutive failures without progress before giving up
)

// DownloadFile downloads an object in ranged chunks, so that a stalled or dropped
// connection, which is likely over a multi-GB model file, resumes from the last byte
// received instead of failing the whole download. It writes to localPath+".part" and
// renames on success, so a failed download never destroys an existing good file.
func (t S3Client) DownloadFile(bucket, key, localPath string) *log.Status {
	err := os.MkdirAll(filepath.Dir(localPath), 0o755)
	if err != nil {
		return log.Error(t.ctx, 500, err, "Could not create path", filepath.Dir(localPath))
	}
	head, err := t.Client.HeadObject(t.ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return log.Error(t.ctx, 500, err, "Could not get object", bucket, key)
	}
	size := aws.ToInt64(head.ContentLength)
	if t.localMatches(localPath, size, aws.ToString(head.ETag)) {
		log.Info(t.ctx, "Local file matches", key, "skipping download", localPath)
		return nil
	}
	log.Info(t.ctx, "Downloading file", key)
	partPath := localPath + ".part"
	f, err := os.Create(partPath)
	if err != nil {
		return log.Error(t.ctx, 500, err, "Could not create file", partPath)
	}
	var offset int64
	attempts := 0
	for offset < size {
		end := min(offset+downloadChunkSize, size) - 1
		n, err := t.downloadRange(bucket, key, f, offset, end)
		offset += n
		if err == nil {
			attempts = 0
			continue
		}
		if n > 0 {
			attempts = 0
		}
		attempts++
		if attempts >= downloadMaxAttempts {
			f.Close()
			os.Remove(partPath)
			return log.Error(t.ctx, 500, err, "Could not copy file", localPath)
		}
		log.Warn(t.ctx, "Download of", key, "interrupted at byte", offset, "of", size,
			"attempt", attempts, "error:", err)
		time.Sleep(time.Duration(attempts) * 2 * time.Second)
		_, err = f.Seek(offset, io.SeekStart)
		if err != nil {
			f.Close()
			os.Remove(partPath)
			return log.Error(t.ctx, 500, err, "Could not seek file", partPath)
		}
	}
	err = f.Close()
	if err != nil {
		os.Remove(partPath)
		return log.Error(t.ctx, 500, err, "Could not close file", partPath)
	}
	err = os.Rename(partPath, localPath)
	if err != nil {
		return log.Error(t.ctx, 500, err, "Could not rename file", partPath, localPath)
	}
	return nil
}

// localMatches reports whether localPath already holds the object's content: the size
// must match, and the MD5 must equal the ETag. The ETag is the MD5 only for objects
// uploaded in a single PutObject, which is how PutFile and PutDirectory upload. A
// multipart ETag ("<hash>-<parts>") cannot be compared, so it never matches, because
// same-size files are common (e.g. model runs of one architecture).
func (t S3Client) localMatches(localPath string, size int64, etag string) bool {
	etag = strings.Trim(etag, `"`)
	if etag == "" || strings.Contains(etag, "-") {
		return false
	}
	info, err := os.Stat(localPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return false
	}
	f, err := os.Open(localPath)
	if err != nil {
		return false
	}
	defer f.Close()
	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), etag)
}

// downloadRange copies bytes start..end (inclusive) of the object to f at its current
// position, and returns the number of bytes written even when it fails part way.
func (t S3Client) downloadRange(bucket, key string, f *os.File, start, end int64) (int64, error) {
	ctx, cancel := context.WithTimeout(t.ctx, downloadChunkTimeout)
	defer cancel()
	rng := "bytes=" + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10)
	out, err := t.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Range:  &rng,
	})
	if err != nil {
		return 0, err
	}
	defer out.Body.Close()
	n, err := io.Copy(f, out.Body)
	if err == nil && n != end-start+1 {
		err = io.ErrUnexpectedEOF
	}
	return n, err
}
