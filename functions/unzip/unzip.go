package main

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/pkg/errors"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Event map[string]interface{}

type S3Object struct {
	id     string
	region string
	bucket string
	key    string
}

func parseEvent(event Event) *S3Object {
	obj := &S3Object{}
	for _, er := range event["Records"].([]interface{}) {
		if region := er.(map[string]interface{})["awsRegion"]; region != nil {
			obj.region = region.(string)
		}
		if s3 := er.(map[string]interface{})["s3"]; s3 != nil {
			obj.bucket = s3.(map[string]interface{})["bucket"].(map[string]interface{})["name"].(string)
			obj.key = s3.(map[string]interface{})["object"].(map[string]interface{})["key"].(string)
		}
	}
	obj.id = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s%s%s", obj.region, obj.bucket, obj.key))))
	return obj
}

func wrapError(err error, message string) error {
	return errors.Wrapf(err, "%s:%s", message, err.Error())
}

func logger(id string, step string, message ...string) {
	log.Printf("[%s]%s %s", id, step, message)
}

func Handler(ctx context.Context, event Event) error {
	// SAM定義から設定した環境変数より解凍先のS3情報の取得
	step := "Init"
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	tgtArn, err := arn.Parse(os.Getenv("TARGET_S3_ARN"))
	log.Print("S3_ENDPOINT:", s3Endpoint)
	log.Printf("TARGET_S3_ARN:%#v", tgtArn)

	// S3イベント情報から必要なものを抽出
	step = "ParseEvent"
	src := parseEvent(event)
	logger(src.id, step, fmt.Sprintf("Source Object: %#v", *src))

	// S3からダウンロードしたファイルを保存するTempファイルを作成
	step = "CreateTempfile"
	tmpfile, err := ioutil.TempFile("/tmp", "srctmp_")
	if err != nil {
		return errors.New(err.Error())
	}
	defer os.Remove(tmpfile.Name())
	logger(src.id, step, fmt.Sprintf("tmpfilename: %s", tmpfile.Name()))

	// ダウンロード処理
	step = "SetClient"
	sess := session.Must(session.NewSession(&aws.Config{
		S3ForcePathStyle: aws.Bool(true),
		Region:           aws.String(src.region),
		Endpoint:         aws.String(s3Endpoint),
	}))
	downloader := s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
		// Option指定
		d.PartSize = 64 * 1024 * 1024
		d.Concurrency = 2
	})

	logger(src.id, step, "Download Start")
	n, err := downloader.Download(
		tmpfile,
		&s3.GetObjectInput{
			Bucket: aws.String(src.bucket),
			Key:    aws.String(src.key),
		})
	if err != nil {
	}
	logger(src.id, step, fmt.Sprintf("%d bytes downloaded", n))

	step = "UnzipTempFile"
	// S3からダウロードしてきたTempファイルを解凍
	r, err := zip.OpenReader(tmpfile.Name())
	if err != nil {
		return wrapError(err, step)
	}
	defer r.Close()
	logger(src.id, step, tmpfile.Name())

	step = "UploadToS3Bucket"
	// 解凍したファイルを宛先のS3にアップロード
	uploader := s3manager.NewUploader(sess)
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return wrapError(err, step)
			}
			defer rc.Close()
			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(tgtArn.Resource),
				Key:    aws.String(f.Name),
				Body:   rc,
			})
			if err != nil {
				return wrapError(err, step)
			}
			logger(src.id, step, fmt.Sprintf("%s uploaded", f.FileInfo().Name()))
		}
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
