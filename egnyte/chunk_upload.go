package egnyte

import (
	"crypto/sha512"
	"encoding/hex"
	"io"
	"sort"
	"sync"
)

type ChunkUploadInfo struct {
	dataMutex      sync.Mutex
	resultMutex    sync.Mutex
	path           string
	chunkNum       int
	data           io.Reader
	remainingBytes int64
	checkSumMap    map[int]string
	chunkSize      int64
	lastChunk      []byte
}

func (c *ChunkUploadInfo) Init(data io.Reader, size int64, chunkSize int64) {
	c.checkSumMap = make(map[int]string)
	c.data = data
	c.remainingBytes = size
	c.chunkSize = chunkSize
	c.chunkNum = 0
	c.lastChunk = nil

}

func (c *ChunkUploadInfo) GetRemainingBytes() int64 {
	c.dataMutex.Lock()
	defer c.dataMutex.Unlock()
	return c.remainingBytes
}

// GetChunk returns chunk
func (c *ChunkUploadInfo) GetChunk() ([]byte, int64, int, error) {
	c.dataMutex.Lock()
	defer c.dataMutex.Unlock()
	if c.remainingBytes == 0 {
		return nil, 0, 0, nil
	}
	c.chunkNum += 1
	buf := make([]byte, c.chunkSize)
	n, err := io.ReadFull(c.data, buf)
	switch err {
	case nil:
		break
	case io.ErrUnexpectedEOF:
		break
	default:
		return nil, 0, 0, err
	}
	c.remainingBytes -= int64(n)
	buf = buf[:n]
	if c.remainingBytes == 0 {
		c.lastChunk = buf
	}
	return buf, c.remainingBytes, c.chunkNum, nil
}

func (c *ChunkUploadInfo) GetLastChunk() ([]byte, int) {
	c.dataMutex.Lock()
	defer c.dataMutex.Unlock()
	return c.lastChunk, c.chunkNum
}

func (c *ChunkUploadInfo) SetChunkCheckSum(chunkNum int, csum string) {
	c.resultMutex.Lock()
	defer c.resultMutex.Unlock()
	c.checkSumMap[chunkNum] = csum
}

// GetResultCsum return final check sum of all chunks
func (c *ChunkUploadInfo) GetResultCsum() string {
	c.resultMutex.Lock()
	defer c.resultMutex.Unlock()
	res := ""
	chunks := make([]int, len(c.checkSumMap))
	i := 0
	for k := range c.checkSumMap {
		chunks[i] = k
		i++
	}
	sort.Ints(chunks)
	for _, chunk := range chunks {
		res += c.checkSumMap[chunk]
	}
	return res
}

// SHA512Digest encode data into SHA512
func SHA512Digest(buf []byte) string {
	csum := sha512.Sum512(buf)
	return hex.EncodeToString(csum[:])
}
