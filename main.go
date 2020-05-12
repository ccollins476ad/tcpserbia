package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func run(device string) error {
	c, err := NewSerConn(device)
	if err != nil {
		return err
	}

	err = processCmds(c)
	if err != nil {
		return err
	}

	return nil
}

func tcpserbiaCmd() *cobra.Command {
	var debug bool
	var device string

	cmd := &cobra.Command{
		Use:   "tcpserbia",
		Short: "XXX",

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				log.SetLevel(log.DebugLevel)
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			if device == "" {
				cmd.HelpFunc()(cmd, args)
				return
			}

			err := run(device)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return
			}
		},
	}

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "", false, "Debug connection")
	cmd.PersistentFlags().StringVarP(&device, "device", "d", "", "Serial device (e.g., /dev/cu.usbserial)")

	return cmd
}

func main() {
	err := tcpserbiaCmd().Execute()
	if err != nil {
		panic(err)
	}
}
