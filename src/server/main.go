package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

const (
	version     = "0.2.0"
	defaultPort = 43187
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "jobdeck:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	command := "serve"
	if len(arguments) > 0 && arguments[0] != "" && arguments[0][0] != '-' {
		command = arguments[0]
		arguments = arguments[1:]
	}

	switch command {
	case "serve":
		return serveCommand(arguments)
	case "import":
		return importCommand(arguments)
	case "status":
		return statusCommand(arguments)
	case "init":
		return initCommand(arguments)
	case "feedback":
		return feedbackCommand(arguments)
	case "record-build":
		return recordBuildCommand(arguments)
	case "verify":
		return verifyCommand(arguments)
	case "version":
		fmt.Printf("Job Deck %s (%s/%s, %s)\n", version, runtime.GOOS,
			runtime.GOARCH, runtime.Version())
		return nil
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func serveCommand(arguments []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	dataPath := flags.String("data", defaultDatabasePath(), "SQLite database path")
	port := flags.Int("port", defaultPort, "fixed loopback port")
	noOpen := flags.Bool("no-open", false, "do not open the browser")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *port < 1024 || *port > 65535 {
		return errors.New("port must be between 1024 and 65535")
	}
	if err := ensurePrivateWorkspace(); err != nil {
		return fmt.Errorf("prepare private workspace: %w", err)
	}

	database, err := openDatabase(*dataPath)
	if err != nil {
		return err
	}
	defer database.Close()

	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(*port))
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		if existingJobDeck(*port) {
			if !*noOpen {
				return openBrowser(dashboardURL(*port))
			}
			return nil
		}
		return fmt.Errorf("port %d is already in use by another application", *port)
	}
	defer listener.Close()

	logger := newActivityLogger(privatePath("logs"))
	application, err := newApplicationServer(database, *port, logger)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Handler:           application.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errChannel := make(chan error, 1)
	go func() {
		errChannel <- httpServer.Serve(listener)
	}()

	url := dashboardURL(*port)
	fmt.Printf("Job Deck %s is ready at %s\n", version, url)
	if err := logger.append("system", "server_started", "Started Job Deck", map[string]string{
		"address": url,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "jobdeck: activity log:", err)
	}
	if !*noOpen {
		if err := openBrowser(url); err != nil {
			fmt.Fprintln(os.Stderr, "jobdeck: open browser:", err)
		}
	}

	signalContext, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownContext)
	case err := <-errChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func importCommand(arguments []string) error {
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	dataPath := flags.String("data", defaultDatabasePath(), "SQLite database path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: jobdeck import [--data PATH] BATCH.json")
	}

	contents, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		return fmt.Errorf("read batch: %w", err)
	}
	var artifact batch
	if err := json.Unmarshal(contents, &artifact); err != nil {
		return fmt.Errorf("decode batch: %w", err)
	}
	database, err := openDatabase(*dataPath)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := importBatch(context.Background(), database, artifact); err != nil {
		return err
	}
	logger := newActivityLogger(privatePath("logs"))
	if err := logger.append("agent", "report_imported", "Imported job report",
		map[string]string{
			"report": artifact.Report.Title,
			"kind":   artifact.Report.Kind,
			"roles":  strconv.Itoa(len(artifact.Jobs)),
		}); err != nil {
		return err
	}
	fmt.Printf("Imported %d jobs into report %s\n", len(artifact.Jobs), artifact.Report.ID)
	return nil
}

func statusCommand(arguments []string) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	dataPath := flags.String("data", defaultDatabasePath(), "SQLite database path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	database, err := openDatabase(*dataPath)
	if err != nil {
		return err
	}
	defer database.Close()
	summary, err := databaseStatus(context.Background(), database)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func initCommand(arguments []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	dataPath := flags.String("data", defaultDatabasePath(), "SQLite database path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if err := ensurePrivateWorkspace(); err != nil {
		return err
	}
	database, err := openDatabase(*dataPath)
	if err != nil {
		return err
	}
	database.Close()
	fmt.Println("Initialized", *dataPath)
	return nil
}

func recordBuildCommand(arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: jobdeck record-build")
	}
	receipt, err := recordBuild(runtime.Version(), runtime.GOOS+"/"+runtime.GOARCH)
	if err != nil {
		return err
	}
	logger := newActivityLogger(privatePath("logs"))
	if err := logger.append("system", "runtime_built", "Built Job Deck locally",
		map[string]string{
			"platform": receipt.Platform,
			"version":  receipt.Version,
		}); err != nil {
		return err
	}
	fmt.Println("Recorded build", receipt.BinaryHash)
	return nil
}

func verifyCommand(arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: jobdeck verify")
	}
	receipt, err := verifyBuild()
	if err != nil {
		return err
	}
	fmt.Printf("Current: Job Deck %s, %s\n", receipt.Version, receipt.Platform)
	return nil
}

func feedbackCommand(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: jobdeck feedback pending|resolve")
	}
	switch arguments[0] {
	case "pending":
		return pendingFeedbackCommand(arguments[1:])
	case "resolve":
		return resolveFeedbackCommand(arguments[1:])
	default:
		return fmt.Errorf("unknown feedback command %q", arguments[0])
	}
}

func existingJobDeck(port int) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	response, err := client.Get(dashboardURL(port) + "/api/health")
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false
	}
	var health struct {
		App string `json:"app"`
	}
	return json.NewDecoder(response.Body).Decode(&health) == nil && health.App == "jobdeck"
}

func dashboardURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func openBrowser(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	return command.Start()
}
