package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/sleeplesslord/saga/internal/webui"
	"github.com/spf13/cobra"
)

var (
	webAddress string
	webNoOpen  bool
	webGlobal  bool
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Open the read-only Saga dashboard",
	Long: `Start a local read-only web interface for visualizing Saga status,
hierarchy, claims, and dependencies. The server binds to localhost only and
uses the same project or global store as the CLI.

All changes continue to happen through Saga CLI commands. The dashboard
automatically refreshes its local snapshot every 30 seconds.`,
	Args: cobra.NoArgs,
	RunE: runWeb,
}

func runWeb(cmd *cobra.Command, _ []string) error {
	st, err := store.New(store.DefaultPath())
	if err != nil {
		return fmt.Errorf("initializing store: %w", err)
	}

	scope := store.ScopeGlobal
	if st.HasLocal() && !webGlobal {
		scope = store.ScopeLocal
	}

	address, err := webui.ListenAddress(webAddress)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("starting web server on %s: %w", address, err)
	}
	defer listener.Close()

	handler, err := webui.Handler(st, webui.Options{
		ProjectName: webui.ProjectName(st),
		Scope:       scope,
	})
	if err != nil {
		return fmt.Errorf("building web interface: %w", err)
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	url := "http://" + listener.Addr().String()
	fmt.Fprintf(cmd.OutOrStdout(), "Saga dispatch board: %s\n", url)
	fmt.Fprintln(cmd.OutOrStdout(), "Read-only view; press Ctrl+C to stop.")

	if !webNoOpen {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Could not open a browser: %v\nOpen %s manually.\n", err, url)
		}
	}

	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(listener) }()
	select {
	case <-cmd.Context().Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("stopping web server: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serving web interface: %w", err)
	}
}

func openBrowser(url string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{url}
	case "windows":
		command = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		command = "xdg-open"
		args = []string{url}
	}
	return exec.Command(command, args...).Start()
}

func init() {
	webCmd.Flags().StringVar(&webAddress, "address", "127.0.0.1:7331", "Local listen address")
	webCmd.Flags().BoolVar(&webNoOpen, "no-open", false, "Do not open the browser automatically")
	webCmd.Flags().BoolVar(&webGlobal, "global", false, "Show the global store instead of the project store")
	rootCmd.AddCommand(webCmd)
}
