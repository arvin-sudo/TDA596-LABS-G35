Installation Steps
Install Protocol Buffers compiler:

Linux
sudo apt install -y protobuf-compiler

macOS
brew install protobuf

Verify installation:

protoc –version

Should show libprotoc 3.x.x or higher
Install Go plugins for protoc:

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1 go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0

Add Go installs to your PATH (add to your shell configuration file):

For bash edit .bashrc, for zsh edit .zshrc:

export PATH="$PATH:$(go env GOPATH)/bin"
Then restart your shell or run source ~/.bashrc (or equivalent).

Set up the project:

Starting in the chord/ directory with Go files and chord.proto:
Initialize the Go module
go mod init chord

Create directory structure
mkdir -p protocol mv chord.proto protocol/

Add gRPC dependencies
go get google.golang.org/grpc@v1.45.0 go get google.golang.org/protobuf@v1.27.1 go mod tidy

Generate the gRPC code:

protoc –go_out=. –go-grpc_out=. –go_opt=module=chord –go-grpc_opt=module=chord protocol/chord.proto

Running the Application
To build:

go build
To create a new Chord ring:

./chord create [-port PORT]
To join an existing Chord ring:

./chord join -addr ADDRESS [-port PORT]