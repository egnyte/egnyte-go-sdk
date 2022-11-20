package egnyte

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"encoding/json"
	"github.com/spf13/cobra"
)

var ClientId string
var Domain string
var Username string
var Password string

// Command will generated config file
// Config file contain auth and refresh token.
var rootCmd = &cobra.Command{
	Use:   "create_config",
	Short: "configuration command will create a config.json",
	Run: func(cmd *cobra.Command, args []string) {
		config := map[string]string{"api_key": ClientId, "username": Username, "password": Password, "domain": Domain}
		token, err := GetAccessToken(context.Background(), config)
		if err != nil {
			fmt.Println(err)
			return
		}
		file, _ := json.MarshalIndent(token, "", " ")

		_ = ioutil.WriteFile("config.json", file, 0644)

	},
}

func init() {
	rootCmd.Flags().StringVarP(&ClientId, "clientId", "c", "", "key received after registering a developer account")
	rootCmd.Flags().StringVarP(&Domain, "domain", "d", "", "Egnyte domain, e.g. example.egnyte.com")
	rootCmd.Flags().StringVarP(&Username, "username", "u", "", "username of Egnyte admin user")
	rootCmd.Flags().StringVarP(&Password, "password", "p", "", "password of the same Egnyte admin user")

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
