package egnyte

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFileSystemContentService_Download(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs-content/Shared/report.pdf", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		if got := r.URL.Query().Get("entry_id"); got != "rev42" {
			t.Errorf("entry_id = %q", got)
		}
		testHeader(t, r, "Range", "bytes=0-3")
		w.Write([]byte("%PDF"))
	})

	body, _, err := client.FileSystemContent.Download(context.Background(), "/Shared/report.pdf", &DownloadOptions{
		EntryID: "rev42",
		Range:   "bytes=0-3",
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "%PDF" {
		t.Errorf("content = %q", data)
	}
}

func TestFileSystemContentService_DownloadByID(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs-content/ids/file/gid123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte("bytes"))
	})

	body, _, err := client.FileSystemContent.DownloadByID(context.Background(), "gid123", nil)
	if err != nil {
		t.Fatalf("DownloadByID: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != "bytes" {
		t.Errorf("content = %q", data)
	}
}

func TestFileSystemContentService_Download_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs-content/Shared/missing.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"File not found"}`))
	})

	body, _, err := client.FileSystemContent.Download(context.Background(), "/Shared/missing.txt", nil)
	if body != nil {
		t.Error("body should be nil on error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Message != "File not found" {
		t.Errorf("APIError = %+v", apiErr)
	}
}

func TestFileSystemContentService_Upload(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs-content/Shared/notes.txt", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "X-Sha512-Checksum", "filehash")
		testHeader(t, r, "Last-Modified", "Sun, 26 Aug 2012 03:55:29 GMT")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf(`FormFile("file"): %v`, err)
		}
		defer file.Close()
		if header.Filename != "notes.txt" {
			t.Errorf("filename = %q", header.Filename)
		}
		data, _ := io.ReadAll(file)
		if string(data) != "hello egnyte" {
			t.Errorf("uploaded content = %q", data)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"path":"/Shared/notes.txt","filename":"notes.txt","checksum":"abc",
			"entry_id":"e1","group_id":"g1","last_modified":"Sun, 26 Aug 2012 03:55:29 GMT","size":12}`))
	})

	result, resp, err := client.FileSystemContent.Upload(context.Background(), "/Shared/notes.txt",
		strings.NewReader("hello egnyte"), &UploadOptions{
			Checksum:     "filehash",
			LastModified: "Sun, 26 Aug 2012 03:55:29 GMT",
		})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	if result.EntryID != "e1" || result.GroupID != "g1" || result.Size != 12 {
		t.Errorf("result = %+v", result)
	}
}

func TestFileSystemContentService_UploadChunk_first(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs-content-chunked/Shared/big.bin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "X-Egnyte-Chunk-Num", "1")
		testHeader(t, r, "X-Egnyte-Chunk-Sha512-Checksum", "chunkhash1")
		testHeader(t, r, "Content-Type", "application/octet-stream")
		if r.Header.Get("X-Egnyte-Upload-Id") != "" || r.Header.Get("X-Egnyte-Last-Chunk") != "" {
			t.Error("first chunk must not send upload id or last-chunk headers")
		}
		// Chunks are raw request bodies, not multipart.
		data, _ := io.ReadAll(r.Body)
		if string(data) != "chunk-1-data" {
			t.Errorf("raw chunk body = %q", data)
		}
		w.Header().Set("X-Egnyte-Upload-Id", "upload-abc")
	})

	upload, _, err := client.FileSystemContent.UploadChunk(context.Background(), "/Shared/big.bin",
		strings.NewReader("chunk-1-data"), ChunkedUploadOptions{ChunkNum: 1, ChunkChecksum: "chunkhash1"})
	if err != nil {
		t.Fatalf("UploadChunk: %v", err)
	}
	if upload.UploadID != "upload-abc" {
		t.Errorf("UploadID = %q, want upload-abc", upload.UploadID)
	}
	if upload.Result != nil {
		t.Error("Result should be nil for a non-final chunk")
	}
}

func TestFileSystemContentService_UploadChunkByID_last(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs-content-chunked/ids/file/gid9", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "X-Egnyte-Chunk-Num", "3")
		testHeader(t, r, "X-Egnyte-Upload-Id", "upload-abc")
		testHeader(t, r, "X-Egnyte-Last-Chunk", "true")
		testHeader(t, r, "X-Sha512-Checksum", "wholefilehash")
		w.Write([]byte(`{"path":"/Shared/big.bin","checksum":"wholefilehash","entry_id":"e9","size":300}`))
	})

	upload, _, err := client.FileSystemContent.UploadChunkByID(context.Background(), "gid9",
		strings.NewReader("chunk-3-data"), ChunkedUploadOptions{
			ChunkNum:      3,
			ChunkChecksum: "chunkhash3",
			UploadID:      "upload-abc",
			LastChunk:     true,
			FileChecksum:  "wholefilehash",
		})
	if err != nil {
		t.Fatalf("UploadChunkByID: %v", err)
	}
	if upload.UploadID != "upload-abc" {
		t.Errorf("UploadID = %q", upload.UploadID)
	}
	if upload.Result == nil || upload.Result.EntryID != "e9" || upload.Result.Size != 300 {
		t.Errorf("Result = %+v", upload.Result)
	}
}

func TestChunkedFileChecksum(t *testing.T) {
	// Format verified against the live API: 2-{numChunks}-{chunkSize}-
	// {sha512 of concatenated hex chunk digests}.
	got := ChunkedFileChecksum(10485760, "aa", "bb")
	digest := func() string {
		h := sha512.Sum512([]byte("aabb"))
		return hex.EncodeToString(h[:])
	}()
	if want := "2-2-10485760-" + digest; got != want {
		t.Errorf("ChunkedFileChecksum = %s, want %s", got, want)
	}
}

func TestFileSystemContentService_UploadChunk_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.FileSystemContent.UploadChunk(ctx, "/x", strings.NewReader("d"),
		ChunkedUploadOptions{ChunkChecksum: "h"}); err == nil {
		t.Error("missing ChunkNum should fail before sending")
	}
	if _, _, err := client.FileSystemContent.UploadChunk(ctx, "/x", strings.NewReader("d"),
		ChunkedUploadOptions{ChunkNum: 1}); err == nil {
		t.Error("missing ChunkChecksum should fail before sending")
	}
	if _, _, err := client.FileSystemContent.UploadChunk(ctx, "/x", nil,
		ChunkedUploadOptions{ChunkNum: 1, ChunkChecksum: "h"}); err == nil {
		t.Error("nil chunk reader should fail before sending")
	}
}

// errReader fails part-way through, simulating a local file that
// becomes unreadable mid-upload.
type errReader struct {
	data []byte
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestFileSystemContentService_Upload_sourceReadError(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs-content/Shared/a.txt", func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Write([]byte(`{}`))
	})

	before := runtime.NumGoroutine()
	want := errors.New("disk went away")
	_, _, err := client.FileSystemContent.Upload(context.Background(), "/Shared/a.txt",
		&errReader{data: []byte("partial"), err: want}, nil)
	if err == nil {
		t.Fatal("a failing content reader should fail the upload")
	}
	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("error = %v, want it to mention %v", err, want)
	}
	assertNoLeakedGoroutines(t, before)
}

func TestFileSystemContentService_Upload_cancelledMidStream(t *testing.T) {
	client, mux := setup(t)
	started := make(chan struct{})
	var once sync.Once
	mux.HandleFunc("/pubapi/v1/fs-content/Shared/big.bin", func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		io.Copy(io.Discard, r.Body)
	})

	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	// A reader that never ends, so the request is still streaming when
	// the context is cancelled.
	endless := io.LimitReader(neverEnding{}, 1<<30)
	go func() {
		<-started
		cancel()
	}()
	_, _, err := client.FileSystemContent.Upload(ctx, "/Shared/big.bin", endless, nil)
	if err == nil {
		t.Fatal("cancelled upload should return an error")
	}
	// The multipart writer goroutine must not outlive the request.
	assertNoLeakedGoroutines(t, before)
}

type neverEnding struct{}

func (neverEnding) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

// assertNoLeakedGoroutines waits briefly for the streaming goroutine to
// unwind, then fails if the count has not returned to its baseline.
func assertNoLeakedGoroutines(t *testing.T, before int) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if runtime.NumGoroutine() <= before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("goroutine leak: %d before, %d after", before, runtime.NumGoroutine())
}

func TestFileSystemContentService_Upload_nilContent(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.FileSystemContent.Upload(context.Background(), "/Shared/a.txt", nil, nil); err == nil {
		t.Error("nil content should fail before sending")
	}
}
