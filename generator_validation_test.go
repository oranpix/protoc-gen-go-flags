package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorRejectsImplicitFlagNameCollision(t *testing.T) {
	runInvalidGeneratorCase(t, "implicit_names.proto", `
syntax = "proto3";
package invalid;

import "flags/annotations.proto";

option go_package = "example.com/invalid;invalid";

message Config {
  string foo_bar = 1 [(flags.value).string = { usage: "foo bar" }];
  string foobar = 2 [(flags.value).string = { usage: "foobar" }];
}
`, "duplicate flag name 'foobar'")
}

func TestGeneratorRejectsNestedShortCollision(t *testing.T) {
	runInvalidGeneratorCase(t, "nested_short.proto", `
syntax = "proto3";
package invalid;

import "flags/annotations.proto";

option go_package = "example.com/invalid;invalid";

message Child {
  string port = 1 [(flags.value).string = { name: "port" short: "p" usage: "child port" }];
}

message Config {
  string parent_port = 1 [(flags.value).string = { name: "parent-port" short: "p" usage: "parent port" }];
  Child child = 2 [(flags.value).message = { nested: true }];
}
`, "duplicate short flag 'p'")
}

func TestGeneratorAllowsNestedShortCollisionWhenChildDisabled(t *testing.T) {
	runValidGeneratorCase(t, "nested_disabled_child_short.proto", `
syntax = "proto3";
package valid;

import "flags/annotations.proto";

option go_package = "example.com/valid;valid";

message DisabledChild {
  option (flags.disabled) = true;

  string port = 1 [(flags.value).string = { name: "port" short: "p" usage: "child port" }];
}

message Config {
  string parent_port = 1 [(flags.value).string = { name: "parent-port" short: "p" usage: "parent port" }];
  DisabledChild child = 2 [(flags.value).message = { nested: true }];
}
`)
}

func TestGeneratorAllowsNestedShortCollisionWhenChildUnexported(t *testing.T) {
	runValidGeneratorCase(t, "nested_unexported_child_short.proto", `
syntax = "proto3";
package valid;

import "flags/annotations.proto";

option go_package = "example.com/valid;valid";

message UnexportedChild {
  option (flags.unexported) = true;
  option (flags.allow_empty) = true;

  string port = 1 [(flags.value).string = { name: "port" short: "p" usage: "child port" }];
}

message Config {
  string parent_port = 1 [(flags.value).string = { name: "parent-port" short: "p" usage: "parent port" }];
  UnexportedChild child = 2 [(flags.value).message = { nested: true }];
}
`)
}

func runInvalidGeneratorCase(t *testing.T, protoName, protoContent, wantErr string) {
	t.Helper()

	out, err := runGeneratorCase(t, protoName, protoContent)
	if err == nil {
		t.Fatalf("buf generate succeeded unexpectedly; output:\n%s", out)
	}
	if !strings.Contains(string(out), wantErr) {
		t.Fatalf("buf generate error = %q, want substring %q", out, wantErr)
	}
}

func runValidGeneratorCase(t *testing.T, protoName, protoContent string) {
	t.Helper()

	out, err := runGeneratorCase(t, protoName, protoContent)
	if err != nil {
		t.Fatalf("buf generate failed: %v\n%s", err, out)
	}
}

func runGeneratorCase(t *testing.T, protoName, protoContent string) ([]byte, error) {
	t.Helper()

	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "protoc-gen-go-flags-test")
	build := exec.Command("go", "build", "-o", pluginPath, ".")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build plugin: %v\n%s", err, out)
	}
	if err := os.Mkdir(filepath.Join(dir, "flags"), 0o700); err != nil {
		t.Fatal(err)
	}
	annotations, err := os.ReadFile(filepath.Join(root, "flags", "annotations.proto"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "flags", "annotations.proto"), string(annotations))
	writeFile(t, filepath.Join(dir, "buf.yaml"), "version: v2\n")
	writeFile(t, filepath.Join(dir, "buf.gen.yaml"), `version: v2
plugins:
  - local: "`+pluginPath+`"
    out: .
    opt:
      - paths=source_relative
`)
	writeFile(t, filepath.Join(dir, protoName), protoContent)

	buf, err := findTool("buf")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(buf, "generate")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+toolPath())
	return cmd.CombinedOutput()
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func toolPath() string {
	paths := []string{os.Getenv("PATH")}
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		paths = append([]string{gobin}, paths...)
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		paths = append([]string{filepath.Join(gopath, "bin")}, paths...)
	}
	if out, err := exec.Command("go", "env", "GOBIN").Output(); err == nil {
		if gobin := strings.TrimSpace(string(out)); gobin != "" {
			paths = append([]string{gobin}, paths...)
		}
	}
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		if gopath := strings.TrimSpace(string(out)); gopath != "" {
			paths = append([]string{filepath.Join(gopath, "bin")}, paths...)
		}
	}
	return strings.Join(paths, string(os.PathListSeparator))
}

func findTool(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	for _, dir := range strings.Split(toolPath(), string(os.PathListSeparator)) {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, name)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}
