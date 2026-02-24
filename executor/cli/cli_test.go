package cli

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSSHCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      Config
		args     []string
		wantCmd  string
		wantArgs []string
	}{
		{
			name: "basic_ssh_with_user_and_key",
			cfg: Config{
				Command: "ffmpeg",
				SSH: SSHConfig{
					Host:    "prod-server.example.com",
					User:    "deploy",
					KeyPath: "/home/deploy/.ssh/id_rsa",
				},
			},
			args:     []string{"-i", "input.mp4", "output.mp4"},
			wantCmd:  "ssh",
			wantArgs: []string{"-l", "deploy", "-i", "/home/deploy/.ssh/id_rsa", "prod-server.example.com", "ffmpeg", "-i", "input.mp4", "output.mp4"},
		},
		{
			name: "ssh_host_only",
			cfg: Config{
				Command: "ls",
				SSH:     SSHConfig{Host: "server.example.com"},
			},
			args:     []string{"-la"},
			wantCmd:  "ssh",
			wantArgs: []string{"server.example.com", "ls", "-la"},
		},
		{
			name: "ssh_with_custom_port",
			cfg: Config{
				Command: "echo",
				SSH: SSHConfig{
					Host: "server.example.com",
					Port: 2222,
					User: "root",
				},
			},
			args:     []string{"hello"},
			wantCmd:  "ssh",
			wantArgs: []string{"-l", "root", "-p", "2222", "server.example.com", "echo", "hello"},
		},
		{
			name: "ssh_with_default_port",
			cfg: Config{
				Command: "echo",
				SSH: SSHConfig{
					Host: "server.example.com",
					Port: 22,
				},
			},
			args:     []string{"hello"},
			wantCmd:  "ssh",
			wantArgs: []string{"server.example.com", "echo", "hello"},
		},
		{
			name: "ssh_with_password",
			cfg: Config{
				Command: "echo",
				SSH: SSHConfig{
					Host:     "server.example.com",
					User:     "admin",
					Password: "secret",
				},
			},
			args:     []string{"hello"},
			wantCmd:  "sshpass",
			wantArgs: []string{"-p", "secret", "ssh", "-l", "admin", "server.example.com", "echo", "hello"},
		},
		{
			name: "ssh_no_args",
			cfg: Config{
				Command: "uptime",
				SSH:     SSHConfig{Host: "server.example.com", User: "deploy"},
			},
			args:     nil,
			wantCmd:  "ssh",
			wantArgs: []string{"-l", "deploy", "server.example.com", "uptime"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := New(tt.cfg, nil, nil)
			defer executor.Close()

			gotCmd, gotArgs := executor.buildSSHCommand(tt.args...)
			assert.Equal(t, tt.wantCmd, gotCmd)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}

func TestExecutor_Start_SSH(t *testing.T) {
	t.Parallel()

	t.Run("ssh_available", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Command: "echo",
			SSH:     SSHConfig{Host: "server.example.com"},
		}

		executor := New(cfg, nil, nil)
		defer executor.Close()

		err := executor.Start()
		require.NoError(t, err)
	})

	t.Run("ssh_with_password_no_sshpass", func(t *testing.T) {
		t.Parallel()

		if _, err := exec.LookPath("sshpass"); err == nil {
			t.Skip("sshpass is installed, skipping negative test")
		}

		cfg := Config{
			Command: "echo",
			SSH: SSHConfig{
				Host:     "server.example.com",
				Password: "secret",
			},
		}

		executor := New(cfg, nil, nil)
		defer executor.Close()

		err := executor.Start()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sshpass not found")
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Command: "echo",
	}

	executor := New(cfg, nil, nil)
	require.NotNil(t, executor)

	t.Cleanup(func() {
		executor.Close()
	})
}

func TestExecutor_Start(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		command     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "success",
			command: "echo",
			wantErr: false,
		},
		{
			name:        "command_not_found",
			command:     "nonexistent_command_12345",
			wantErr:     true,
			errContains: `"nonexistent_command_12345" not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := Config{
				Command: tt.command,
			}

			executor := New(cfg, nil, nil)
			t.Cleanup(func() {
				executor.Close()
			})

			err := executor.Start()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecutor_Close(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Command: "echo",
	}

	executor := New(cfg, nil, nil)

	err := executor.Close()
	require.NoError(t, err)

	err = executor.Close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "executor is already closed")
}

func TestExecutor_Run_Closed(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Command: "echo",
	}

	executor := New(cfg, nil, nil)

	err := executor.Close()
	require.NoError(t, err)

	ctx := context.Background()
	err = executor.Execute(ctx, "test")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "executor is closed")
}
