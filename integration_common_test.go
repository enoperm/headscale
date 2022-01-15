//go:build integration
// +build integration

package headscale

import (
	"bytes"
	"fmt"
	"time"

	"inet.af/netaddr"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const DOCKER_EXECUTE_TIMEOUT = 10 * time.Second

var IpPrefix4 = netaddr.MustParseIPPrefix("100.64.0.0/10")
var IpPrefix6 = netaddr.MustParseIPPrefix("fd7a:115c:a1e0::/48")

type ExecuteCommandConfig struct {
	timeout time.Duration
}

type ExecuteCommandOption func(*ExecuteCommandConfig) error

func ExecuteCommandTimeout(timeout time.Duration) ExecuteCommandOption {
	return ExecuteCommandOption(func(conf *ExecuteCommandConfig) error {
		conf.timeout = timeout
		return nil
	})
}

func ExecuteCommand(
	resource *dockertest.Resource,
	cmd []string,
	env []string,
	options ...ExecuteCommandOption,
) (string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	execConfig := ExecuteCommandConfig{
		timeout: DOCKER_EXECUTE_TIMEOUT,
	}

	for _, opt := range options {
		if err := opt(&execConfig); err != nil {
			return "", fmt.Errorf("execute-command/options: %w", err)
		}
	}

	type result struct {
		exitCode int
		err      error
	}

	resultChan := make(chan result, 1)

	// Run your long running function in it's own goroutine and pass back it's
	// response into our channel.
	go func() {
		exitCode, err := resource.Exec(
			cmd,
			dockertest.ExecOptions{
				Env:    append(env, "HEADSCALE_LOG_LEVEL=disabled"),
				StdOut: &stdout,
				StdErr: &stderr,
			},
		)
		resultChan <- result{exitCode, err}
	}()

	// Listen on our channel AND a timeout channel - which ever happens first.
	select {
	case res := <-resultChan:
		if res.err != nil {
			return "", res.err
		}

		if res.exitCode != 0 {
			fmt.Println("Command: ", cmd)
			fmt.Println("stdout: ", stdout.String())
			fmt.Println("stderr: ", stderr.String())

			return "", fmt.Errorf("command failed with: %s", stderr.String())
		}

		return stdout.String(), nil
	case <-time.After(execConfig.timeout):

		return "", fmt.Errorf("command timed out after %s", execConfig.timeout)
	}
}

func DockerRestartPolicy(config *docker.HostConfig) {
	// set AutoRemove to true so that stopped container goes away by itself on error *immediately*.
	// when set to false, containers remain until the end of the integration test.
	config.AutoRemove = false
	config.RestartPolicy = docker.RestartPolicy{
		Name: "no",
	}
}

func DockerAllowLocalIPv6(config *docker.HostConfig) {
	if config.Sysctls == nil {
		config.Sysctls = make(map[string]string, 1)
	}
	config.Sysctls["net.ipv6.conf.all.disable_ipv6"] = "0"
}

func DockerAllowNetworkAdministration(config *docker.HostConfig) {
	config.CapAdd = append(config.CapAdd, "NET_ADMIN")
	config.Mounts = append(config.Mounts, docker.HostMount{
		Type:   "bind",
		Source: "/dev/net/tun",
		Target: "/dev/net/tun",
	})
}
