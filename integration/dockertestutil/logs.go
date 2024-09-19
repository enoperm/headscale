package dockertestutil

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const filePerm = 0o644

func WriteLog(
	pool *dockertest.Pool,
	resource *dockertest.Resource,
	stdout io.Writer,
	stderr io.Writer,
) error {
	return pool.Client.Logs(
		docker.LogsOptions{
			Context:      context.TODO(),
			Container:    resource.Container.ID,
			OutputStream: stdout,
			ErrorStream:  stderr,
			Tail:         "all",
			RawTerminal:  false,
			Stdout:       true,
			Stderr:       true,
			Follow:       false,
			Timestamps:   false,
		},
	)
}

func SaveLog(
	pool *dockertest.Pool,
	resource *dockertest.Resource,
	basePath string,
) (string, string, error) {
	err := os.MkdirAll(basePath, os.ModePerm)
	if err != nil {
		return "", "", err
	}

	log.Printf("Saving logs for %s to %s\n", resource.Container.Name, basePath)

	stdoutPath := path.Join(basePath, resource.Container.Name+".stdout.log")
	stdout, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return "", "", fmt.Errorf("failed to open stdout for writing: %w", err)
	}
	defer stdout.Close()

	stderrPath := path.Join(basePath, resource.Container.Name+".stderr.log")
	stderr, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return "", "", fmt.Errorf("failed to open stderr for writing: %w", err)
	}
	defer stderr.Close()

	bufOut := bufio.NewWriter(stdout)
	bufErr := bufio.NewWriter(stderr)

	defer bufOut.Flush()
	defer bufErr.Flush()

	err = WriteLog(pool, resource, bufOut, bufErr)
	if err != nil {
		return "", "", err
	}

	return stdoutPath, stderrPath, nil
}
