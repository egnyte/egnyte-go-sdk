package egnyte

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestSalesforceService_FolderMap(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/sfdc/foldermap/001A000001BcDeFGHI", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"path":"Accounts/ABC","folder_id":"a74bb119"}`))
	})

	folderMap, _, err := client.Salesforce.FolderMap(context.Background(), "001A000001BcDeFGHI")
	if err != nil {
		t.Fatalf("FolderMap: %v", err)
	}
	if folderMap.Path != "Accounts/ABC" || folderMap.FolderID != "a74bb119" {
		t.Errorf("folderMap = %+v", folderMap)
	}
}

func TestSalesforceService_FolderMap_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.Salesforce.FolderMap(context.Background(), "tooshort"); err == nil {
		t.Error("non-18-character record id should fail before sending")
	}
}

func TestSalesforceService_FolderMap_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/sfdc/foldermap/001A000001BcDeFGHI", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Mapping not found"}`))
	})

	_, _, err := client.Salesforce.FolderMap(context.Background(), "001A000001BcDeFGHI")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("APIError = %+v", apiErr)
	}
}
