# gorpc
A lightweight tool for generating Go code from `.proto` files.  
It automates the creation of `.proto` files from templates and compiles them using `protoc` with Go-specific options.

## Features

- Automatically detects `.proto` files in the specified directory.
- Creates new `.proto` files from a customizable template if none exist.
- Uses `protoc` to generate Go and gRPC code with proper options.

## Getting Started

### Prerequisites

- Go 1.16 or later (for `//go:embed` support).
- Protocol Buffers Compiler (`protoc`) installed and available in your `PATH`.  
[protobuf/releases](https://github.com/protocolbuffers/protobuf/releases/)  
Example for protoc-30.2-linux-x86_64:
```sh
wget 'https://github.com/protocolbuffers/protobuf/releases/download/v30.2/protoc-30.2-linux-x86_64.zip'
ls -lh
rm -rf $HOME/protoc/
mkdir -p $HOME/protoc
unzip protoc-30.2-linux-x86_64.zip -d $HOME/protoc
$HOME/protoc/bin/protoc --version
file $HOME/protoc/bin/protoc
sudo ln -s  $HOME/protoc/bin/protoc /usr/bin/protoc
which protoc
# /usr/bin/protoc
file $(which protoc)
protoc --version
```

- `protoc-gen-go` and `protoc-gen-go-grpc` plugins installed. You can install them using:
```sh
go install -x -ldflags=-s google.golang.org/protobuf/cmd/protoc-gen-go@latest
protoc-gen-go --version

go install -x -ldflags=-s google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
protoc-gen-go-grpc --version
```

## Installation
```sh
git clone git@github.com:wasmup/gorpc.git
cd gorpc
CGO_ENABLED=0 GOOS=linux GOAMD64=v3 go install -x -ldflags "-s -w -linkmode internal" -trimpath=true 
which gorpc
file $(which gorpc)
ls -lh $(which gorpc)
```

## Usage

```sh
cd /path/to/your/proto/files/
gorpc

# or
gorpc /path/to/your/proto/files/
```

If no `.proto` files are found in the directory, a new `.proto` file will be created using the embedded template.

## Example
```sh
mkdir myservice
cd myservice

# generate  `myservice.proto`
gorpc

# build `myservice.pb.go`
gorpc
```

If no `.proto` files exist, a new file (e.g., `myservice.proto`) will be created based on the template.

The tool will automatically compile the `.proto` files using protoc and generate Go code.

## Template Customization
The `.proto` template is embedded in the binary using Go's `//go:embed` directive. You can modify the `event.txt` file to customize the template.

Example template (event.txt):
```proto
edition = "2023"; // successor to proto2 and proto3
// syntax = "proto3";

package {{.Name}};
option features.(pb.go).api_level = API_OPAQUE;
// option go_package = ".;{{.Name}}";

import "google/protobuf/timestamp.proto";
import "google/protobuf/go_features.proto";

message Event {
  string name = 1;
  string id = 2;
  google.protobuf.Timestamp created_at = 3;
}
```

## License
This project is licensed under the MIT License. See the LICENSE file for details.
