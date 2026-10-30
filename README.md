# gorpc

`gorpc` generates Go protobuf and gRPC source files from `.proto` files. If a directory contains no schemas, it creates a starter event schema instead.

## Requirements and installation

- Go 1.27.3 or newer (as specified by `go.mod`).
- `protoc` with support for protobuf edition 2023, including the `google/protobuf` definitions and `go_features.proto` needed by the starter schema.
- `protoc-gen-go` and `protoc-gen-go-grpc` on `PATH`, compatible with the schemas and `protoc` version in use.

Install the generator and plugins:

```sh
go install github.com/wasmup/gorpc@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Make sure Go's binary directory (usually `$(go env GOPATH)/bin`) is on `PATH`. Install `protoc` separately using your platform's package manager or the protobuf project's releases.

## Usage

```sh
gorpc                       # current directory
gorpc ./schemas ./other     # one or more directories
```

The command searches each directory recursively for `.proto` files and generates source-relative `.pb.go` and `_grpc.pb.go` files alongside them. It invokes `protoc` with the directory as its working directory, so schema imports must be resolvable from there or from the standard protobuf include paths. The command supplies Go import mappings based on the directory name. When there are no `.proto` files, it writes a starter schema named after the directory, replacing characters that are invalid in protobuf identifiers with underscores. Run the command again to generate Go code from that schema. Existing output files may be overwritten by `protoc`.

A failure in any directory or `protoc` invocation results in a nonzero exit status; remaining directories and files are still processed.

## Examples

The `examples/client` and `examples/server` directories are separate Go modules demonstrating a local gRPC call. Run `go run .` in the server directory, then `go run .` in the client directory. The server listens on port 50051 without transport encryption, for local demonstration only.

`examples/producer-consumer` demonstrates protobuf messages with Kafka and requires a Kafka broker on `localhost:9092`, CGO, and the dependencies described in its README.

## License

MIT; see [LICENSE](LICENSE).
