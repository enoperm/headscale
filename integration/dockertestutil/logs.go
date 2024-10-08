package dockertestutil

import (
	"bytes"
	"context"
	"log"
	"os"
	"path"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const filePerm = 0o644

func SaveLog(
	pool *dockertest.Pool,
	resource *dockertest.Resource,
	basePath string,
) (string, string, error) {
	err := os.MkdirAll(basePath, os.ModePerm)
	if err != nil {
		return "", "", err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err = pool.Client.Logs(
		docker.LogsOptions{
			Context:      context.TODO(),
			Container:    resource.Container.ID,
			OutputStream: &stdout,
			ErrorStream:  &stderr,
			Tail:         "all",
			RawTerminal:  false,
			Stdout:       true,
			Stderr:       true,
			Follow:       false,
			Timestamps:   false,
		},
	)
	if err != nil {
		return "", "", err
	}

	log.Printf("Saving logs for %s to %s\n", resource.Container.Name, basePath)

	stdoutPath := path.Join(basePath, resource.Container.Name+".stdout.log")
	err = os.WriteFile(
		stdoutPath,
		stdout.Bytes(),
		filePerm,
	)
	if err != nil {
		return "", "", err
	}

	stderrPath := path.Join(basePath, resource.Container.Name+".stderr.log")
	err = os.WriteFile(
		stderrPath,
		stderr.Bytes(),
		filePerm,
	)
	if err != nil {
		return "", "", err
	}

	return stdoutPath, stderrPath, nil
}
