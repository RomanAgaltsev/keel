package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel/internal/config"
)

func newConfigCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{Use: "config", Short: "Manage keel's user config"}
	cmd.PersistentFlags().StringVar(&file, "file", "", "config file path (default: user config dir)")

	resolve := func() (string, error) {
		if file != "" {
			return file, nil
		}
		return config.Path()
	}

	get := &cobra.Command{
		Use:   "get <key>",
		Short: "Print a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolve()
			if err != nil {
				return err
			}
			c, err := config.LoadFrom(p)
			if err != nil {
				return err
			}
			v, err := getField(c, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), v)
			return nil
		},
	}

	set := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolve()
			if err != nil {
				return err
			}
			c, err := config.LoadFrom(p)
			if err != nil {
				return err
			}
			if err := setField(&c, args[0], args[1]); err != nil {
				return err
			}
			return config.SaveTo(p, c)
		},
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "Print all config values",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := resolve()
			if err != nil {
				return err
			}
			c, err := config.LoadFrom(p)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "author.name=%s\nauthor.email=%s\nprovider=%s\n", c.AuthorName, c.AuthorEmail, c.Provider)
			return nil
		},
	}

	cmd.AddCommand(get, set, list)
	return cmd
}

func getField(c config.Config, key string) (string, error) {
	switch key {
	case "author.name":
		return c.AuthorName, nil
	case "author.email":
		return c.AuthorEmail, nil
	case "provider":
		return c.Provider, nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func setField(c *config.Config, key, val string) error {
	switch key {
	case "author.name":
		c.AuthorName = val
	case "author.email":
		c.AuthorEmail = val
	case "provider":
		c.Provider = val
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}
