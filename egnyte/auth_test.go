package egnyte

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var Config map[string]string

func init() {
	dirname, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("error in finding config.")
	}
	filePath := path.Join(dirname, ".egnyte", "test_config.json")
	file, _ := os.OpenFile(filePath, os.O_RDWR, 666)
	byteValue, _ := ioutil.ReadAll(file)

	// we initialize our Users array

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &Config)
	fmt.Println(Config)
	/* load test data */
	Config["RootPath"] = "/Shared/test/"

}

// TestGetAccessToken
func TestGetAccessToken(t *testing.T) {
	if _, ok := Config["accessToken"]; ok {
		resp, err := GetAccessToken(context.Background(), Config)
		fmt.Println(resp)
		if err != nil {
			t.Errorf("%s", err)
		}
		if !resp.Valid() {
			t.Errorf("%s", resp)
		}
	}
}
