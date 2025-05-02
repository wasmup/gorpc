
```sh
nc -zv localhost 9092

go build -tags musl ./...

CC=musl-gcc CGO_ENABLED=1 go build -ldflags '-linkmode external -extldflags "-static -s -w"' -tags musl .
which musl-gcc
apk add musl-dev musl-tools

ls -l
file ./app

```
