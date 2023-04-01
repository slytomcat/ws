package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

// Version is app version
const Version = "0.2.2"

var options struct {
	origin       string
	printVersion bool
	insecure     bool
	subProtocals string
	timestamp    bool
	binAsText    bool
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "ws URL",
		Short: fmt.Sprintf("websocket client v.%s", Version),
		Run:   root,
	}
	rootCmd.Flags().StringVarP(&options.origin, "origin", "o", "", "websocket origin")
	rootCmd.Flags().BoolVarP(&options.printVersion, "version", "v", false, "print version")
	rootCmd.Flags().BoolVarP(&options.insecure, "insecure", "k", false, "skip ssl certificate check")
	rootCmd.Flags().StringVarP(&options.subProtocals, "subprotocal", "s", "", "sec-websocket-protocal field")
	rootCmd.Flags().BoolVarP(&options.timestamp, "timestamp", "t", false, "print timestamps for sent and incoming messages")
	rootCmd.Flags().BoolVarP(&options.binAsText, "bin2text", "b", false, "print binary message as text")

	rootCmd.Execute()
}

func root(cmd *cobra.Command, args []string) {
	if options.printVersion {
		fmt.Printf("ws v.%s\n", Version)
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
		if dest.Scheme == "wss" {
			originURL.Scheme = "https"
		} else {
			originURL.Scheme = "http"
		}
		options.origin = originURL.String()
	}

	var historyFile string
	user, err := user.Current()
	if err == nil {
		historyFile = filepath.Join(user.HomeDir, ".ws_history")
	}

	err = connect(dest.String(), &readline.Config{
		Prompt:      "> ",
		HistoryFile: historyFile,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if err != io.EOF && err != readline.ErrInterrupt {
			os.Exit(1)
		}
	}
}
