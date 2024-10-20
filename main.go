package main

import (
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"time"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

var (
	version = "local build"
	options struct {
		origin       string
		printVersion bool
		insecure     bool
		subProtocals string
		initMsg      string
		authHeader   string
		timestamp    bool
		binAsText    bool
		pingPong     bool
		compression  bool
		pingInterval time.Duration
		filter       *regexp.Regexp
	}
	filter string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ws URL",
		Short: fmt.Sprintf("ws is a websocket client v.%s", version),
		Run:   root,
	}
	rootCmd.Flags().StringVarP(&options.origin, "origin", "o", "", "websocket origin (default value is formed from URL)")
	rootCmd.Flags().BoolVarP(&options.printVersion, "version", "v", false, "print version")
	rootCmd.Flags().BoolVarP(&options.insecure, "insecure", "k", false, "skip ssl certificate check")
	rootCmd.Flags().StringVarP(&options.subProtocals, "subprotocal", "s", "", "sec-websocket-protocal field")
	rootCmd.Flags().StringVarP(&options.authHeader, "auth", "a", "", "auth header value, like 'Bearer $TOKEN'")
	rootCmd.Flags().BoolVarP(&options.timestamp, "timestamp", "t", false, "print timestamps for sent and received messages")
	rootCmd.Flags().BoolVarP(&options.binAsText, "bin2text", "b", false, "print binary message as text")
	rootCmd.Flags().BoolVarP(&options.pingPong, "pingPong", "p", false, "print out ping/pong messages")
	rootCmd.Flags().DurationVarP(&options.pingInterval, "interval", "i", 0, "send ping each interval (ex: 20s)")
	rootCmd.Flags().StringVarP(&options.initMsg, "init", "m", "", "connection init message")
	rootCmd.Flags().BoolVarP(&options.compression, "compression", "c", false, "enable compression")
	rootCmd.Flags().StringVarP(&filter, "filter", "f", "", "only messages that match regexp will be printed")
	rootCmd.Execute()
}

func root(cmd *cobra.Command, args []string) {
	if options.printVersion {
		fmt.Printf("ws v.%s\n", version)
		os.Exit(0)
	}
	if len(args) != 1 {
		cmd.Help()
		os.Exit(1)
	}
	dest, err := url.Parse(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if options.origin == "" {
		originURL := *dest
		switch dest.Scheme {
		case "wss":
			originURL.Scheme = "https"
		case "ws":
			originURL.Scheme = "http"
		default:
			fmt.Printf("unsupported scheme: %s\n", dest.Scheme)
			os.Exit(1)
		}
		options.origin = originURL.String()
	}
	if len(filter) > 0 {
		options.filter, err = regexp.Compile(filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "compiling regexp '%s' error: %v", filter, err)
			os.Exit(1)
		}
	}
	var historyFile string
	user, err := user.Current()
	if err == nil {
		historyFile = filepath.Join(user.HomeDir, ".ws_history")
	}
	rl, err := readline.NewEx(&readline.Config{
		Prompt:      "> ",
		HistoryFile: historyFile,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := &Session{rl: rl}
	errs := s.connect(dest.String())
	if len(errs) > 0 {
		fmt.Println()
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
