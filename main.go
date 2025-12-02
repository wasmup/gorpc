package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	_ "embed"
	_ "time/tzdata"

	"github.com/klauspost/cpuid/v2"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	})))

	slog.Info(`starting gorpc`,
		`Version`, runtime.Version(),
		`OS`, runtime.GOOS,
		`ARCH`, runtime.GOARCH,
		`GOAMD64`, GOAMD64(),
		// `Cardinality`, Cardinality(),
		`now`, time.Now(),
		`Local`, time.Local)

	dirs := os.Args[1:]
	if len(dirs) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			slog.Error(`Getwd`, `Err`, err)
			return
		}
		dirs = []string{cwd}
	}

	for _, dir := range dirs {
		if err := os.Chdir(dir); err != nil {
			slog.Error(`msg`, `Err`, err)
			return
		}
		name := filepath.Base(dir)
		if name == "" {
			continue
		}

		protoFiles, err := GetProtoFilesInCurrentDir(dir)
		if err != nil {
			slog.Error(`msg`, `Err`, err)
			continue
		}

		if len(protoFiles) == 0 {
			CreateNewProroFileFrom(proto, name)
			return
		}

		for _, file := range protoFiles {
			// Build protoc command arguments
			args := []string{
				"--go_out=.",
				"--go-grpc_out=require_unimplemented_servers=false:.",
				fmt.Sprintf("--go_opt=M%s=./%s", file, name),
				fmt.Sprintf("--go-grpc_opt=M%s=./%s", file, name),
				"--go_opt=paths=source_relative",
				"--go-grpc_opt=paths=source_relative",
				file,
			}

			fmt.Println("\nprotoc", strings.Join(args, " "))

			// Create command
			cmd := exec.Command("protoc", args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Execute command
			if err := cmd.Run(); err != nil {
				slog.Error(`protoc execution failed`, `Err`, err)
				continue
			}

			fmt.Printf("\nProtobuf %q generated successfully in package %q \n", file, name)
		}
	}
}

func GetProtoFilesInCurrentDir(root string) (protoFiles []string, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".proto" {
			name := filepath.Base(path)
			if name != "" {
				protoFiles = append(protoFiles, name)
			}
		}
		return nil
	})
	return
}

//go:embed event.txt
var proto string

// This function is used to create a new file with the given name and write the content to it.
// It uses the text/template package to parse the content and execute it with the provided name.
// The resulting file will be named <name>.proto and will be created in the current working directory.
func CreateNewProroFileFrom(content, filename string) {
	// Create a new template and parse the content
	t, err := template.New("proto").Parse(content)
	if err != nil {
		slog.Error("ParseTemplate", "Err", err)
		return
	}

	// Create the output file
	file, err := os.Create(filename + ".proto")
	if err != nil {
		slog.Error("Create", "Err", err)
		return
	}
	defer file.Close()

	// Execute the template with the provided name
	data := map[string]string{
		"Name": filename,
	}
	if err := t.Execute(file, data); err != nil {
		slog.Error("ExecuteTemplate", "Err", err)
		return
	}

	slog.Info("Proto file created successfully", "File", filename+".proto")
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
	var start = time.Now()

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
