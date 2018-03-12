package main

import (
	"encoding/json"
	"testing"
)

var (
	dummyS3Event = `
{
  "Records": [
    {
      "eventVersion": "2.0",
      "eventTime": "1970-01-01T00:00:00.000Z",
      "requestParameters": {
        "sourceIPAddress": "127.0.0.1"
      },
      "s3": {
        "configurationId": "testConfigRule",
        "object": {
          "eTag": "0123456789abcdef0123456789abcdef",
          "sequencer": "0A1B2C3D4E5F678901",
          "key": "upload",
          "size": 1024
        },
        "bucket": {
          "arn": "arn:aws:s3:::zipfiles",
          "name": "zipfiles",
          "ownerIdentity": {
            "principalId": "EXAMPLE"
          }
        },
        "s3SchemaVersion": "1.0"
      },
      "responseElements": {
        "x-amz-id-2": "EXAMPLE123/5678abcdefghijklambdaisawesome/mnopqrstuvwxyzABCDEFGH",
        "x-amz-request-id": "EXAMPLE123456789"
      },
      "awsRegion": "ap-northeast-1",
      "eventName": "ObjectCreated:Put",
      "userIdentity": {
        "principalId": "EXAMPLE"
      },
      "eventSource": "aws:s3"
    }
  ]
}
`
)

func TestParse(t *testing.T) {
	var ev Event
	err := json.Unmarshal([]byte(dummyS3Event), &ev)
	if err != nil {
		t.Error(err)
	}
	src := parseEvent(ev)

	if src.bucket != "zipfiles" {
		t.Errorf("parsed s3 bucket [%s] invalid", src.bucket)
	}

	if src.key != "upload" {
		t.Errorf("parsed s3 key [%s] invalid", src.key)
	}

	if src.region != "ap-northeast-1" {
		t.Error("parsed s3 region [%s] invalid", src.region)
	}

}
