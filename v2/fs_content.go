package egnyte

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

// FileSystemContentService handles downloading and uploading file contents.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/file-system-management
// OAuth scope: Egnyte.filesystem. Access to individual items follows the
// authenticated user's file and folder permissions.
type FileSystemContentService service

// UploadResult is the metadata returned after a successful upload.
type UploadResult struct {
	Path         string `json:"path"`
	Filename     string `json:"filename"`
	Checksum     string `json:"checksum"`
	EntryID      string `json:"entry_id"`
	GroupID      string `json:"group_id"`
	LastModified string `json:"last_modified"`
	Size         int64  `json:"size"`
}

// DownloadOptions control content downloads.
type DownloadOptions struct {
	// EntryID downloads a specific revision instead of the latest.
	EntryID string
	// Range sets an HTTP Range header for partial content retrieval,
	// e.g. "bytes=0-1023".
	Range string
}

// UploadOptions control simple (non-chunked) uploads.
type UploadOptions struct {
	// Checksum is the SHA512 hash of the entire file, used by the server
	// to validate upload integrity (X-Sha512-Checksum).
	Checksum string
	// LastModified sets the file's last-modified timestamp, e.g.
	// "Sun, 26 Aug 2012 03:55:29 GMT". Defaults to the current time.
	LastModified string
}

// ChunkedUploadOptions control one chunk of a chunked upload. Files larger
// than 100 MB should be uploaded in chunks of equal size (10 MB – 1 GB,
// 100 MB recommended); only the last chunk may be smaller. Chunks expire
// 24 hours after the first chunk is uploaded.
type ChunkedUploadOptions struct {
	// ChunkNum is the 1-based chunk number. Required.
	ChunkNum int
	// ChunkChecksum is the hex-encoded SHA512 hash of this chunk's
	// data. Required.
	ChunkChecksum string
	// UploadID is the id returned by the server for the first chunk.
	// Required for every chunk after the first.
	UploadID string
	// LastChunk marks the final chunk; the server then assembles the file.
	LastChunk bool
	// FileChecksum is the whole-file checksum required on the final
	// chunk (X-Sha512-Checksum). For chunked uploads this is NOT a
	// plain SHA512 of the file bytes — compute it with
	// ChunkedFileChecksum from the per-chunk checksums.
	FileChecksum string
	// LastModified optionally sets the file's last-modified timestamp.
	LastModified string
}

// ChunkedUpload is the outcome of uploading one chunk. After the first
// chunk, UploadID carries the id to pass with subsequent chunks. Result is
// non-nil only once the final chunk has been processed.
type ChunkedUpload struct {
	UploadID string
	Result   *UploadResult
}

// ChunkedFileChecksum derives the whole-file checksum Egnyte expects on
// the final chunk of a chunked upload (X-Sha512-Checksum). The format,
// validated against the live API, is
//
//	2-{numChunks}-{chunkSize}-{digest}
//
// where 2 is a format version, chunkSize is the size in bytes of the
// standard (non-final) chunks, and digest is the SHA512 of the
// concatenated hex-encoded SHA512 digests of every chunk in order. A
// plain SHA512 of the file bytes is rejected. This is also the format
// of the checksum the API reports for assembled chunked files.
func ChunkedFileChecksum(chunkSize int64, chunkChecksums ...string) string {
	h := sha512.New()
	for _, c := range chunkChecksums {
		h.Write([]byte(c))
	}
	return fmt.Sprintf("2-%d-%d-%s", len(chunkChecksums), chunkSize, hex.EncodeToString(h.Sum(nil)))
}

// Download returns the contents of the file at path as a stream. The
// caller must close the returned ReadCloser.
//
// GET /pubapi/v1/fs-content/{path}
func (s *FileSystemContentService) Download(ctx context.Context, filePath string, opts *DownloadOptions) (io.ReadCloser, *Response, error) {
	return s.download(ctx, "v1/fs-content"+EncodePath(ensureLeadingSlash(filePath)), opts)
}

// DownloadByID returns the contents of the file with the given file
// system id (group_id) as a stream. The caller must close the returned
// ReadCloser.
//
// GET /pubapi/v1/fs-content/ids/file/{id}
func (s *FileSystemContentService) DownloadByID(ctx context.Context, id string, opts *DownloadOptions) (io.ReadCloser, *Response, error) {
	return s.download(ctx, "v1/fs-content/ids/file/"+url.PathEscape(id), opts)
}

func (s *FileSystemContentService) download(ctx context.Context, urlPath string, opts *DownloadOptions) (io.ReadCloser, *Response, error) {
	if opts != nil && opts.EntryID != "" {
		urlPath += "?" + url.Values{"entry_id": {opts.EntryID}}.Encode()
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	if opts != nil && opts.Range != "" {
		req.Header.Set("Range", opts.Range)
	}
	return s.client.DoRaw(req)
}

// Upload stores content as the file at path, creating it or overwriting
// an existing file. Intermediate folders must already exist. For files
// larger than 100 MB use UploadChunk instead.
//
// POST /pubapi/v1/fs-content/{path}
func (s *FileSystemContentService) Upload(ctx context.Context, filePath string, content io.Reader, opts *UploadOptions) (*UploadResult, *Response, error) {
	if content == nil {
		return nil, nil, fmt.Errorf("egnyte: upload content must not be nil")
	}
	req, err := s.newMultipartRequest(ctx, "v1/fs-content"+EncodePath(ensureLeadingSlash(filePath)), path.Base(filePath), content)
	if err != nil {
		return nil, nil, err
	}
	if opts != nil {
		if opts.Checksum != "" {
			req.Header.Set("X-Sha512-Checksum", opts.Checksum)
		}
		if opts.LastModified != "" {
			req.Header.Set("Last-Modified", opts.LastModified)
		}
	}
	result := new(UploadResult)
	resp, err := s.client.Do(req, result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// UploadChunk uploads one chunk of a chunked upload targeting the file at
// path. Chunks are sent as the raw request body (the spec describes
// multipart, but the live API's chunked endpoint takes raw bytes). See
// ChunkedUploadOptions for the chunk protocol.
//
// POST /pubapi/v1/fs-content-chunked/{path}
func (s *FileSystemContentService) UploadChunk(ctx context.Context, filePath string, chunk io.Reader, opts ChunkedUploadOptions) (*ChunkedUpload, *Response, error) {
	return s.uploadChunk(ctx, "v1/fs-content-chunked"+EncodePath(ensureLeadingSlash(filePath)), chunk, opts)
}

// UploadChunkByID uploads one chunk of a chunked upload targeting the
// existing file with the given file system id (group_id).
//
// POST /pubapi/v1/fs-content-chunked/ids/file/{id}
func (s *FileSystemContentService) UploadChunkByID(ctx context.Context, id string, chunk io.Reader, opts ChunkedUploadOptions) (*ChunkedUpload, *Response, error) {
	return s.uploadChunk(ctx, "v1/fs-content-chunked/ids/file/"+url.PathEscape(id), chunk, opts)
}

func (s *FileSystemContentService) uploadChunk(ctx context.Context, urlPath string, chunk io.Reader, opts ChunkedUploadOptions) (*ChunkedUpload, *Response, error) {
	if chunk == nil {
		return nil, nil, fmt.Errorf("egnyte: chunk reader must not be nil")
	}
	if opts.ChunkNum < 1 {
		return nil, nil, fmt.Errorf("egnyte: chunk number must be >= 1")
	}
	if opts.ChunkChecksum == "" {
		return nil, nil, fmt.Errorf("egnyte: chunk checksum is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Body = io.NopCloser(chunk)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Egnyte-Chunk-Num", strconv.Itoa(opts.ChunkNum))
	req.Header.Set("X-Egnyte-Chunk-Sha512-Checksum", opts.ChunkChecksum)
	if opts.UploadID != "" {
		req.Header.Set("X-Egnyte-Upload-Id", opts.UploadID)
	}
	if opts.LastChunk {
		req.Header.Set("X-Egnyte-Last-Chunk", "true")
	}
	if opts.FileChecksum != "" {
		req.Header.Set("X-Sha512-Checksum", opts.FileChecksum)
	}
	if opts.LastModified != "" {
		req.Header.Set("Last-Modified", opts.LastModified)
	}

	upload := new(ChunkedUpload)
	var resp *Response
	if opts.LastChunk {
		upload.Result = new(UploadResult)
		resp, err = s.client.Do(req, upload.Result)
	} else {
		resp, err = s.client.Do(req, nil)
	}
	if err != nil {
		return nil, resp, err
	}
	upload.UploadID = resp.Header.Get("X-Egnyte-Upload-Id")
	if upload.UploadID == "" {
		upload.UploadID = opts.UploadID
	}
	return upload, resp, nil
}

// newMultipartRequest builds a multipart/form-data POST with a single
// "file" part whose bytes are streamed from content, so large files never
// need to be buffered in memory.
func (s *FileSystemContentService) newMultipartRequest(ctx context.Context, urlPath, filename string, content io.Reader) (*http.Request, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile("file", filename)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, content); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.CloseWithError(mw.Close())
	}()

	req, err := s.client.NewRequest(ctx, http.MethodPost, urlPath, nil)
	if err != nil {
		pr.Close()
		return nil, err
	}
	req.Body = pr
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req, nil
}
