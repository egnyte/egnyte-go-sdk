package egnyte

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// Egnyte returns the modification time for a file as a string in the LastModifiedStr
// variable. parseFileModTime parses this string into a time object and stores it in
// the ModTime variable of the file object
func (o *Object) parseFileModTime() error {
	timeStr := strings.TrimSuffix(o.LastModifiedStr, " GMT")
	parsedTime, err := time.Parse(TimeFormat, timeStr)
	if err != nil {
		return err
	}
	o.ModTime = parsedTime
	return nil
}

// Egnyte returns the modification time for a folder as epoch int in the LastModifiedEpoch
// variable. parseFolderModTime parses this int into a time object and stores it in
// the ModTime variable of the folder object
func (o *Object) parseFolderModTime() {
	parsedTime := time.Unix(o.LastModifiedEpoch/1000, 0)
	o.ModTime = parsedTime
}

// parseModTime parses the mod time for the file or folder object and stores it into
// ModTime variable of the object
func (o *Object) parseModTime() error {
	if o.IsFolder {
		o.parseFolderModTime()
	} else {
		err := o.parseFileModTime()
		if err != nil {
			return err
		}
	}

	for _, folder := range o.Folders {
		folder.parseFolderModTime()
	}

	for _, file := range o.Files {
		err := file.parseFileModTime()
		if err != nil {
			return err
		}
	}

	for _, file := range o.Versions {
		err := file.parseFileModTime()
		if err != nil {
			return err
		}
	}

	return nil
}

// creates a file
func (o *Object) createFile(ctx context.Context) (*Object, error) {
	uri := fmt.Sprintf(URI_GET_FILE, o.Path)
	modTime := o.ModTime.Format(ModTimeLayout)
	opts := &requestOptions{
		Method: "POST",
		Path:   uri,
		Body:   o.Body,
		ExtraHeaders: map[string]string{
			"Last-Modified": modTime,
		},
	}

	resp, err := o.Client.doRequest(ctx, opts, nil, nil)
	if err != nil {
		return nil, err
	}

	retObject := &Object{
		ModTime:  o.ModTime,
		Checksum: resp.Header.Get("X-Sha512-Checksum"),
		Etag:     resp.Header.Get("Etag"),
		Path:     o.Path,
	}
	return retObject, nil
}

// creates a folder
func (o *Object) createFolder(ctx context.Context) (*Object, error) {
	uri := fmt.Sprintf(URI_CREATE_FOLDER, o.Path)
	req := createFolderRequest{
		Action: "add_folder",
	}
	opts := &requestOptions{
		Method: "POST",
		Path:   uri,
	}
	var newObject *Object
	_, err := o.Client.doRequest(ctx, opts, &req, &newObject)
	if err != nil {
		return nil, err
	}
	newObject.IsFolder = true
	return newObject, nil
}

// creates a file or folder
func (o *Object) Create(ctx context.Context) (*Object, error) {
	if o.IsFolder {
		return o.createFolder(ctx)
	} else {
		return o.createFile(ctx)
	}
}

// Downloads a file
func (o *Object) Get(ctx context.Context) (io.ReadCloser, error) {
	uri := fmt.Sprintf(URI_GET_FILE, o.Path)
	params := url.Values{}
	if o.EntryID != "" {
		params.Set("entry_id", o.EntryID)
	}
	opts := &requestOptions{
		Method:        "GET",
		Path:          uri,
		DontCloseBody: true,
		Parameters:    params,
	}
	resp, err := o.Client.doRequest(ctx, opts, nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// TODO: Moves a file or folder object from it's current path to the newPath
func (o *Object) Move(ctx context.Context, newPath string) error {
	return nil
}

// deletes a file or folder
func (o *Object) Delete(ctx context.Context) error {
	uri := fmt.Sprintf(URI_DELETE_OBJECT, o.Path)
	params := url.Values{}
	if o.EntryID != "" {
		params.Set("entry_id", o.EntryID)
	}
	opts := &requestOptions{
		Method:     "DELETE",
		Path:       uri,
		Parameters: params,
	}
	_, err := o.Client.doRequest(ctx, opts, nil, nil)
	return err
}

// TODO: Copies a file or folder object to a new path
func (o *Object) Copy(ctx context.Context, newPath string) error {
	return nil
}

// lists a file or folder
func (o *Object) List(ctx context.Context) (*Object, error) {
	url := fmt.Sprintf(URI_LIST, o.Path)
	opts := &requestOptions{
		Method: "GET",
		Path:   url,
	}
	var list *Object
	_, err := o.Client.doRequest(ctx, opts, nil, &list)
	if err != nil {
		return nil, err
	}
	err = list.parseModTime()
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (o *Object) ChunkUpload(ctx context.Context, uploadInfo *UploadInfo, extraHeaders map[string]string) error {
	uri := fmt.Sprintf(URI_CHUNKED_UPLOAD, o.Path)
	opts := &requestOptions{
		Method:       "POST",
		Path:         uri,
		ExtraHeaders: extraHeaders,
		Body:         uploadInfo.Data,
	}
	resp, err := o.Client.doRequest(ctx, opts, nil, nil)
	if err != nil {
		return err
	}
	if uploadInfo.UploadID == "" {
		uploadInfo.UploadID = resp.Header.Get("X-Egnyte-Upload-Id")
	}
	return nil
}

// TODO: Fetch information about folder size and item counts (for both file version
// and files inside the folder and all the subfolders)
func (o *Object) Stats(ctx context.Context) error {
	return nil
}

// TODO: Lock a file
func (o *Object) Lock(ctx context.Context) error {
	return nil
}

// TODO: Unlock a file
func (o *Object) Unlock(ctx context.Context) error {
	return nil
}
