module github.com/wemall/media-service

go 1.21

require (
	github.com/aws/aws-sdk-go-v2 v1.26.1
	github.com/aws/aws-sdk-go-v2/config v1.27.11
	github.com/aws/aws-sdk-go-v2/service/s3 v1.53.1
	github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign v1.7.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/wemall/gen v0.0.0
	github.com/wemall/pkg v0.0.0
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.1
)

replace github.com/wemall/gen => ../../gen

replace github.com/wemall/pkg => ../../pkg
