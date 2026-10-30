package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDirectoriesAndNestedFiles(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"first/nested", "second"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"first/nested/a.proto", "second/b.proto"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("syntax = \"proto3\";"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	// Record the working directory and arguments without requiring protoc plugins.
	script := "#!/bin/sh\nprintf '%s|%s\\n' \"$PWD\" \"$*\" >> \"$RECORD\"\n"
	if err := os.WriteFile(filepath.Join(bin, "protoc"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	record := filepath.Join(root, "record")
	t.Setenv("RECORD", record)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := run([]string{"first", "second"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, filepath.Join(root, "first")+"|") || !strings.Contains(got, "nested/a.proto") || !strings.Contains(got, filepath.Join(root, "second")+"|") || !strings.Contains(got, "b.proto") {
		t.Fatalf("unexpected calls: %s", got)
	}
}

func TestRunCreatesValidStarterAndReportsErrors(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "123-my-service")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{dir}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "_123_my_service.proto"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "package _123_my_service;") {
		t.Fatalf("invalid package: %s", data)
	}
	if err := run([]string{filepath.Join(root, "missing")}); err == nil {
		t.Fatal("missing directory succeeded")
	}
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "protoc"), []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if err := run([]string{dir}); err == nil {
		t.Fatal("failed protoc succeeded")
	}
}
