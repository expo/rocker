package tests

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/grammarly/rocker/src/test"
	"github.com/kr/pretty"
	"github.com/mitchellh/go-homedir"
)

type rockerBuildOptions struct {
	Rockerfile    string
	GlobalOptions []string
	BuildOptions  []string
	Wd            string
	Stdout        io.Writer
}

func runCmd(executable string, stdoutWriter io.Writer /* stderr io.Writer,*/, params ...string) error {
	return runCmdWithWd(executable, "", stdoutWriter, params...)
}

func runCmdWithWd(executable, wd string, stdoutWriter io.Writer /* stderr io.Writer,*/, params ...string) error {
	cmd := exec.Command(executable, params...)

	if *verbosityLevel >= 1 {
		fmt.Printf("Running: %v\n", strings.Join(cmd.Args, " "))
	}

	cmd.Dir = wd

	if stdoutWriter != nil {
		cmd.Stdout = stdoutWriter
	}

	if *verbosityLevel >= 2 {
		if cmd.Stdout == nil {
			// If there was no stdout writer assigned
			cmd.Stdout = os.Stdout
		} else if cmd.Stdout != os.Stdout {
			// If there was stdout writer assigned but was not os.Stdout
			cmd.Stdout = io.MultiWriter(os.Stdout, stdoutWriter)
		}
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func removeImage(imageName string) error {
	return runCmd("docker", nil, "rmi", imageName)
}

func getImageShaByName(imageName string) (string, error) {
	var b bytes.Buffer

	if err := runCmd("docker", &b, "images", "-q", imageName); err != nil {
		fmt.Println("Can't execute command:", err)
		return "", err
	}

	sha := strings.Trim(b.String(), "\n")

	if len(sha) < 12 {
		return "", fmt.Errorf("Too short sha (should be at least 12 chars) got: %q", sha)
	}

	//fmt.Printf("Image: %v, size: %d\n", sha, len(sha))

	return sha, nil
}
func getRockerBinaryPath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		panic("$GOPATH is not defined")
	}
	return gopath + "/bin/rocker"
}

func runRockerPull(image string) error {
	if err := runCmd(getRockerBinaryPath(), nil, "pull", image); err != nil {
		return err
	}

	return nil
}
func runRockerWithFile(filename string) error {
	if err := runCmd(getRockerBinaryPath(), nil, "build", "--no-cache", "-f", filename); err != nil {
		return err
	}

	return nil
}

func createTempFile(content string) (string, error) {
	tmpfile, err := ioutil.TempFile("/tmp/", "rocker_integration_test_")
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}

func runRockerBuildWithFile(filename string, opts ...string) error {
	p := []string{"build", "-f", filename}
	params := append(p, opts...)

	if err := runCmd(getRockerBinaryPath(), nil, params...); err != nil {
		return err
	}

	return nil
}
func runRockerBuildWithOptions(content string, opts ...string) error {
	filename, err := createTempFile(content)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filename)

	p := []string{"build", "-f", filename}
	params := append(p, opts...)
	if err := runCmd(getRockerBinaryPath(), nil, params...); err != nil {
		return err
	}

	return nil
}
func runRockerBuildWdWithOptions(wd string, opts ...string) error {
	if *verbosityLevel >= 2 {
		fmt.Printf("CWD: %s\n", wd)
	}

	p := []string{"build"}
	params := append(p, opts...)
	if err := runCmdWithWd(getRockerBinaryPath(), wd, nil, params...); err != nil {
		return err
	}

	return nil
}
func runRockerBuild(content string) error {
	return runRockerBuildWithOptions(content)
}

func runRockerBuildWithOptions2(opts rockerBuildOptions) error {
	filename, err := createTempFile(opts.Rockerfile)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filename)

	opts1 := append(opts.GlobalOptions, "build", "-f", filename)
	opts1 = append(opts1, opts.BuildOptions...)

	return runCmdWithWd(getRockerBinaryPath(), opts.Wd, opts.Stdout, opts1...)
}

func makeTempDir(t *testing.T, prefix string, files map[string]string) string {
	// We produce tmp dirs within home to make integration tests work within
	// Mac OS and VirtualBox
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}

	baseTmpDir := path.Join(home, ".rocker-integ-tmp")

	if err := os.MkdirAll(baseTmpDir, 0755); err != nil {
		log.Fatal(err)
	}

	tmpDir, err := ioutil.TempDir(baseTmpDir, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if err := test.MakeFiles(tmpDir, files); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}
	if *verbosityLevel >= 2 {
		fmt.Printf("temp directory: %s\n", tmpDir)
		fmt.Printf("  with files: %# v\n", pretty.Formatter(files))
	}
	return tmpDir
}

func debugf(format string, args ...interface{}) {
	if *verbosityLevel >= 2 {
		fmt.Printf(format, args...)
	}
}
