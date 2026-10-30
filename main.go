package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"

	_ "embed"
	_ "time/tzdata"

	"github.com/klauspost/cpuid/v2"
	"github.com/prometheus/client_golang/prometheus"
)

/*
generate new folder and just output go code into new folder:
protoc --go_out=. --go-grpc_out=. echo.proto

If you only need protobuf marshaling/unmarshaling (no gRPC), then:
protoc --go_out=. echo.proto

Generates:

text
echo.pb.go
This includes:

Message structs
Enums
Reflection metadata
Marshal / Unmarshal support
Size calculation
Proto cloning, comparison, etc.

You don’t need --go-grpc_out if:

You’re sending protobuf over:
raw TCP
HTTP
Kafka
NATS
files
shared memory
You’re storing protobuf on disk
You’re using protobuf as an internal binary format
*/
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo})))
	if err := run(os.Args[1:]); err != nil {
		slog.Error("generation failed", "error", err)
		os.Exit(1)
	}
}

// run generates Go files for each directory, or creates a starter schema in an empty directory.
func run(dirs []string) error {
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	var failures []error
	for _, dir := range dirs {
		root, err := filepath.Abs(dir)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		name := protoName(filepath.Base(root))
		files, err := GetProtoFilesInCurrentDir(root)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", dir, err))
			continue
		}
		if len(files) == 0 {
			if err := createProtoFile(proto, root, name); err != nil {
				failures = append(failures, err)
			}
			continue
		}
		for _, file := range files {
			args := []string{
				"--go_out=.", "--go-grpc_out=require_unimplemented_servers=false:.",
				fmt.Sprintf("--go_opt=M%s=./%s", file, name),
				fmt.Sprintf("--go-grpc_opt=M%s=./%s", file, name),
				"--go_opt=paths=source_relative", "--go-grpc_opt=paths=source_relative", file,
			}
			fmt.Println("protoc", strings.Join(args, " "))
			cmd := exec.Command("protoc", args...)
			cmd.Dir = root
			cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				failures = append(failures, fmt.Errorf("%s: protoc: %w", filepath.Join(root, file), err))
			}
		}
	}
	return errors.Join(failures...)
}

// protoName converts a directory name into a valid protobuf identifier.
func protoName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	result := b.String()
	if result == "" {
		return "event"
	}
	if unicode.IsDigit(rune(result[0])) {
		return "_" + result
	}
	return result
}

// GetProtoFilesInCurrentDir returns paths relative to root, including nested schemas.
func GetProtoFilesInCurrentDir(root string) (protoFiles []string, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".proto" {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			protoFiles = append(protoFiles, filepath.ToSlash(rel))
		}
		return nil
	})
	return
}

//go:embed event.txt
var proto string

func createProtoFile(content, root, name string) error {
	t, err := template.New("proto").Parse(content)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	path := filepath.Join(root, name+".proto")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := t.Execute(file, map[string]string{"Name": name}); err != nil {
		file.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	slog.Info("Proto file created successfully", "file", path)
	return nil
}

func GOAMD64() string {
	level := `v1`
	if cpuid.CPU.Supports(cpuid.SSE3, cpuid.SSSE3, cpuid.SSE4, cpuid.SSE42, cpuid.POPCNT, cpuid.LAHF) {
		level = `v2`
	}
	if cpuid.CPU.Supports(cpuid.AVX, cpuid.AVX2, cpuid.BMI1, cpuid.BMI2, cpuid.FMA3, cpuid.LZCNT, cpuid.MOVBE, cpuid.F16C) {
		level = `v3`
	}
	if cpuid.CPU.Supports(cpuid.AVX512F, cpuid.AVX512BW, cpuid.AVX512CD, cpuid.AVX512DQ, cpuid.AVX512VL) {
		level = `v4`
	}
	return level
}

func Cardinality() (total int) {
	start := time.Now()

	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		slog.Error(`metricsCardinality failed`, `d`, time.Since(start), `error`, err)
		return
	}

	for _, mf := range metricFamilies {
		cardinality := len(mf.GetMetric())
		total += cardinality
	}

	return
}
