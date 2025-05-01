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
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo})))
	slog.Info(`Go`, `Version`, runtime.Version(), `OS`, runtime.GOOS, `ARCH`, runtime.GOARCH, `now`, time.Now(), `Local`, time.Local)

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
