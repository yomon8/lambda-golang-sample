PROJECT_NAME      := "lambdagolangsample"
STAGING_S3_BUCKET := "otomo-devel"
STAGING_S3_PREFIX := "package"
STACK_NAME        := "my-stack"

all: build

build: test
	for d in $$(ls -d ./functions/*);do GOOS=linux GOARCH=amd64 go build $${d};done

integrationtest: build 
	docker-compose down
	zip -r testdata.zip ./test/data
	docker-compose -p $(PROJECT_NAME) up -d
	while true;do  if [ $$(docker-compose logs | grep "Ready" | wc -l) -gt 0 ];then break;else echo "waiting...";sleep 1;fi;done
	aws --endpoint-url=http://localhost:4572 s3 mb s3://zipfiles
	aws --endpoint-url=http://localhost:4572 s3 mb s3://unzippedfiles
	aws --endpoint-url=http://localhost:4572 s3 cp testdata.zip s3://zipfiles/testdata.zip
	aws-sam-local local generate-event s3 --region ap-northeast-1 --bucket zipfiles --key testdata.zip > ./test/env/s3event.json
	aws-sam-local local invoke Unzip -e ./test/env/s3event.json --env-vars ./test/env/sam-local-env.json \
		--docker-network $$(docker network ls -q -f name=$(PROJECT_NAME))
	aws --endpoint-url=http://localhost:4572 s3 cp s3://unzippedfiles/test/data/data.txt - | cmp test/data/data.txt -
	docker-compose -p $(PROJECT_NAME) down

test: 
	go test ./functions/...
	
deploy:
	aws-sam-local package --s3-bucket $(STAGING_S3_BUCKET) --s3-prefix $(STAGING_S3_PREFIX) --template-file ./template.yaml --output-template-file  ./packaged.yaml
	aws-sam-local deploy --template-file ./packaged.yaml --stack-name $(STACK_NAME) --capabilities CAPABILITY_IAM
