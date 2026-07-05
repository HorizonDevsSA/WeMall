module github.com/wemall/chat-service

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/nats-io/nats.go v1.33.1
	github.com/sqlc-dev/pqtype v0.3.0
	github.com/wemall/gen v0.0.0
	github.com/wemall/pkg v0.0.0
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.1
)

require (
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240415180920-8c6c420018be // indirect
)

replace (
	github.com/wemall/gen => ../../gen
	github.com/wemall/pkg => ../../pkg
)
