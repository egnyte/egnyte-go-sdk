package egnyte

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"os"
	"path"
	"testing"
)

var accessToken string

// Test Create folder
func TestCreateFolder(t *testing.T) {
	fmt.Println(Config)
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	uuid, _ := uuid.NewUUID()
	obj := Object{
		Client:   client,
		Path:     path.Join(Config["RootPath"], fmt.Sprintf("%v", uuid)),
		IsFolder: true,
	}
	dstObj, err := obj.Create(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}

	if dstObj == nil {
		t.Errorf("%s", err)
	}
	Config["DestinationFolderPath"] = path.Join(Config["RootPath"], fmt.Sprintf("%v", uuid))
}

// Test Create new file
func TestCreateFile(t *testing.T) {

	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	uuid, _ := uuid.NewUUID()
	tempPath := path.Join(os.TempDir(), fmt.Sprintf("%v", uuid))
	emptyFile, _ := os.Create(tempPath)
	emptyFile.Close()
	in, err := os.OpenFile(tempPath, os.O_RDWR, 0666)
	fileInfo, err := in.Stat()

	obj := Object{
		Client:  client,
		Path:    path.Join(Config["RootPath"], fmt.Sprintf("%v", uuid)),
		Body:    in,
		Size:    int(fileInfo.Size()),
		ModTime: fileInfo.ModTime(),
	}
	dstObj, err := obj.Create(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}

	if dstObj == nil {
		t.Errorf("%s", err)
	}
	Config["DestinationFilePath"] = path.Join(Config["RootPath"], fmt.Sprintf("%v", uuid))

}

// Test Delete folder
func TestDeleteFolder(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	obj := Object{
		Client:   client,
		Path:     Config["DestinationFolderPath"],
		IsFolder: true,
	}
	err = obj.Delete(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}
}

// Test Get List Of File In Folder
func TestGetListOfFileInFolder(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	obj := Object{
		Client:   client,
		Path:     "/Shared/",
		IsFolder: true,
	}
	dir, err := obj.List(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}
	if dir == nil {
		t.Errorf("%+v", dir)
	}

	for _, file := range dir.Files {
		println(file.Path)
	}
}

// Test Get List Of Folders In Folder
func TestGetListOfFoldersInFolder(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	obj := Object{
		Client:   client,
		Path:     "/Shared/",
		IsFolder: true,
	}
	dir, err := obj.List(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}
	if dir == nil {
		t.Errorf("%+v", dir)
	}

	for _, file := range dir.Folders {
		println(file.Path)
	}
}

// Test Download File
func TestDownloadFile(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	obj := Object{
		Client: client,
		Path:   Config["DestinationFilePath"],
	}
	resp, err := obj.Get(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}
	if resp == nil {
		t.Errorf("%s", err)
	}
}

// Test Delete File
func TestDeleteFile(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	obj := Object{
		Client: client,
		Path:   Config["DestinationFilePath"],
	}
	err = obj.Delete(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}
}
