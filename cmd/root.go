// Copyright © 2017 uxbh

// Package cmd implments the ztdns command-line interface.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "ztdns",
	Short: "Zerotier DNS Server",
	Long: `ztDNS is a dedicated DNS server for ZeroTier networks.
This application will serve DNS requests for the members of a ZeroTier
network for both A (IPv4) and AAAA (IPv6) requests`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().Bool("debug", false, "enable debug messages")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ztdns.yml)")
	viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug"))

}

func initConfig() {
	if cfgFile != "" {
		// Use specified config file
		viper.SetConfigFile(cfgFile)
	} else {
		// Find config file in current directory or $HOME
		viper.SetConfigName("ztdns")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
	}

	// Enable setting config values with ZTDNS_KEY environment variables
	viper.SetEnvPrefix("ztdns")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}
