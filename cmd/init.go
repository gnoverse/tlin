package cmd

import (
	"fmt"

	"os"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/gnolang/tlin/lint"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// initCmd: tlin init
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new linter configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfigurationFile(cfgFile); err != nil {
			logger.Error("Error initializing config file", zap.Error(err))
			return
		}
		fmt.Printf("Configuration file created/updated: %s\n", cfgFile)
	},
}

func initConfigurationFile(configurationPath string) error {
	if configurationPath == "" {
		configurationPath = ".tlin.yaml"
	}

	// Create a yaml file with rules
	config := lint.Config{
		Name:  "tlin",
		Rules: map[string]tt.ConfigRule{},
	}
	d, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	f, err := os.Create(configurationPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(d)
	if err != nil {
		return err
	}

	return nil
}
