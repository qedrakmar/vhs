package main

import (
	"log"

	"github.com/gramLabs/vhs/capture"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "vhs",
		Short: "A tool for capturing and recording network traffic.",
	}

	captureResponse bool

	address    string
	middleware string
	protocol   string
)

func main() {
	rootCmd.PersistentFlags().BoolVar(&captureResponse, "capture-response", false, "Capture the responses.")
	rootCmd.PersistentFlags().StringVar(&address, "address", capture.DefaultAddr, "Address VHS will use to capture traffic.")
	rootCmd.PersistentFlags().StringVar(&middleware, "middleware", "", "A path to an executable that VHS will use as middleware.")
	rootCmd.PersistentFlags().StringVar(&protocol, "protocol", "http", "Protocol to be used when assembling packets.")

	rootCmd.AddCommand(recordCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}