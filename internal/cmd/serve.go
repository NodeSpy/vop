package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/NodeSpy/vop/internal/credserver"
	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve [start|stop|status]",
		Short: "Manage the ECS-compatible credential endpoint",
		Long: `Start a local HTTP server that implements the ECS container credential
provider protocol. AWS SDKs natively support this via the
AWS_CONTAINER_CREDENTIALS_FULL_URI environment variable.

Subcommands:
  start   Start the server in the foreground (default if omitted)
  stop    Stop a running server
  status  Show whether the server is running and which profiles are loaded

Use -d to run as a background daemon instead.

The server accepts credentials pushed from vop shell, exec, and auth.
Containers fetch them via GET /creds/<profile>.`,
		Args: cobra.MaximumNArgs(1),
		RunE: cmdServe,
	}
	cmd.Flags().IntP("port", "p", 41920, "Port to listen on")
	cmd.Flags().String("token", "", "Authorization token (random if not set)")
	cmd.Flags().BoolP("detach", "d", false, "Run as a background daemon")
	// Hidden flag used internally when daemonizing: the re-exec'd child
	// runs with --background-child so it knows to start the server directly.
	cmd.Flags().Bool("background-child", false, "")
	_ = cmd.Flags().MarkHidden("background-child")
	return cmd
}

func cmdServe(cmd *cobra.Command, args []string) error {
	sub := "start"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "start":
		return cmdServeStart(cmd)
	case "stop":
		return cmdServeStop()
	case "status":
		return cmdServeStatus()
	default:
		return fmt.Errorf("unknown subcommand: %s (use start, stop, or status)", sub)
	}
}

func cmdServeStart(cmd *cobra.Command) error {
	port, _ := cmd.Flags().GetInt("port")
	token, _ := cmd.Flags().GetString("token")
	detach, _ := cmd.Flags().GetBool("detach")
	bgChild, _ := cmd.Flags().GetBool("background-child")

	// Check if already running.
	if info := credserver.LoadServerInfo(); info != nil {
		client := credserver.NewClientFrom(info.Port, info.Token)
		if client.Ping() {
			ui.Info("Server already running on port %d (PID %d).", info.Port, info.PID)
			return nil
		}
		// Stale PID file.
		credserver.RemoveServerInfo()
	}

	// Daemon mode (-d): re-exec ourselves as a background child.
	if detach && !bgChild {
		return daemonize(port, token)
	}

	// Foreground (default) or background-child: actually start the server.
	return runServer(port, token)
}

// runServer starts the credential server and blocks until it shuts down.
func runServer(port int, token string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	names := c.ProfileNames()
	if len(names) == 0 {
		return fmt.Errorf("no profiles configured. Run 'vop add' to create one")
	}

	srv, err := credserver.NewServer(c, nil, token, port)
	if err != nil {
		return fmt.Errorf("failed to start credential server: %w", err)
	}

	// Write server info so other commands can find us.
	if err := credserver.SaveServerInfo(&credserver.ServerInfo{
		PID:   os.Getpid(),
		Port:  port,
		Token: srv.AuthToken(),
	}); err != nil {
		return fmt.Errorf("failed to save server info: %w", err)
	}

	ui.Success("Credential server listening on port %d (PID %d)", port, os.Getpid())
	ui.Info("Push credentials with: vop auth <profile>")
	ui.Info("Stop with:             Ctrl-C or vop serve stop")
	fmt.Println()

	// Handle shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println()
		ui.Info("Shutting down...")
		credserver.RemoveServerInfo()
		_ = srv.Close()
	}()

	if srvErr := srv.Serve(); srvErr != nil && srvErr.Error() != "http: Server closed" {
		credserver.RemoveServerInfo()
		return srvErr
	}

	return nil
}

func cmdServeStop() error {
	info := credserver.LoadServerInfo()
	if info == nil {
		ui.Info("No credential server is running.")
		return nil
	}

	p, err := os.FindProcess(info.PID)
	if err != nil {
		credserver.RemoveServerInfo()
		ui.Info("No credential server is running.")
		return nil
	}

	if err := p.Signal(syscall.SIGTERM); err != nil {
		credserver.RemoveServerInfo()
		return fmt.Errorf("failed to stop server (PID %d): %w", info.PID, err)
	}

	credserver.RemoveServerInfo()
	ui.Success("Credential server stopped (PID %d).", info.PID)
	return nil
}

func cmdServeStatus() error {
	info := credserver.LoadServerInfo()
	if info == nil {
		ui.Info("No credential server is running.")
		return nil
	}

	client := credserver.NewClientFrom(info.Port, info.Token)
	if !client.Ping() {
		credserver.RemoveServerInfo()
		ui.Info("No credential server is running (stale PID file cleaned up).")
		return nil
	}

	fmt.Println()
	ui.Success("Credential server running on port %d (PID %d)", info.Port, info.PID)
	fmt.Println()

	ui.Info("Docker environment (add to your docker-compose.yml service):")
	fmt.Println()

	c, err := loadConfig()
	if err == nil {
		for _, name := range c.ProfileNames() {
			fmt.Printf("  # profile: %s\n", name)
			fmt.Println("  environment:")
			fmt.Printf("    - AWS_CONTAINER_CREDENTIALS_FULL_URI=http://host.docker.internal:%d/creds/%s\n", info.Port, name)
			fmt.Printf("    - AWS_CONTAINER_AUTHORIZATION_TOKEN=%s\n", info.Token)
			fmt.Println()
		}
	}

	return nil
}

// daemonize re-execs the current binary with --background-child in the background.
func daemonize(port int, token string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable path: %w", err)
	}

	args := []string{"serve", "start", "--background-child", "--port", fmt.Sprintf("%d", port)}
	if token != "" {
		args = append(args, "--token", token)
	}

	child := exec.Command(exe, args...)
	child.Stdout = nil
	child.Stderr = nil
	child.Stdin = nil
	// Detach from the parent process group so the child survives.
	setSysProcAttr(child)

	if err := child.Start(); err != nil {
		return fmt.Errorf("failed to start background server: %w", err)
	}

	// Wait briefly for the server to write its info file.
	// The child writes it after binding the port.
	for i := 0; i < 20; i++ {
		if info := credserver.LoadServerInfo(); info != nil {
			fmt.Println()
			ui.Success("Credential server started on port %d (PID %d)", info.Port, info.PID)
			fmt.Println()
			ui.Info("Push credentials with: vop auth <profile>")
			ui.Info("Stop with:             vop serve stop")
			ui.Info("Check status:          vop serve status")
			fmt.Println()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	ui.Warn("Server process started (PID %d) but info file not yet written.", child.Process.Pid)
	return nil
}

// ensureServer starts the credential server as a background daemon if it
// is not already running. Used by shell --serve.
func ensureServer(cmd *cobra.Command) error {
	if info := credserver.LoadServerInfo(); info != nil {
		client := credserver.NewClientFrom(info.Port, info.Token)
		if client.Ping() {
			return nil // Already running
		}
		credserver.RemoveServerInfo()
	}

	port := 41920
	token := ""
	if cmd.Flags().Changed("port") {
		port, _ = cmd.Flags().GetInt("port")
	}
	if cmd.Flags().Changed("token") {
		token, _ = cmd.Flags().GetString("token")
	}

	return daemonize(port, token)
}
