package template

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/http_range"
	"github.com/alist-org/alist/v3/pkg/utils"
)

type NotionService struct {
	cookie     string
	token      string
	spaceID    string
	databaseID string
	filePageID string
	userId     string
}

type FileInfo struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	SHA1         string    `json:"sha1"`
	NotionPageID string    `json:"notion_page_id"`
	URL          string    `json:"url"`
	ContentType  string    `json:"content_type"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Directory struct {
	ID         int       `json:"id" gorm:"primaryKey"`
	Name       string    `json:"name"`
	ParentID   *int      `json:"parent_id" gorm:"index"`
	DatabaseID string    `json:"database_id" gorm:"index"`
	Deleted    bool      `json:"deleted" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type File struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	SHA1         string    `json:"sha1" gorm:"index"`
	NotionPageID string    `json:"notion_page_id"`
	DirectoryID  int       `json:"directory_id" gorm:"index"`
	IsChunked    bool      `json:"is_chunked" gorm:"default:false"`
	ChunkSize    int64     `json:"chunk_size" gorm:"default:0"`
	Deleted      bool      `json:"deleted" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FileChunk 存储文件分块信息
type FileChunk struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	FileID       int       `json:"file_id" gorm:"index"`
	ChunkIndex   int       `json:"chunk_index"`
	ChunkSize    int64     `json:"chunk_size"`
	StartOffset  int64     `json:"start_offset"`
	EndOffset    int64     `json:"end_offset"`
	NotionPageID string    `json:"notion_page_id"`
	SHA1         string    `json:"sha1"`
	Deleted      bool      `json:"deleted" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type NotionFile struct {
	URL        string `json:"url"`
	ExpiryTime string `json:"expiry_time"`
}

type PropertyResponse struct {
	Object string       `json:"object"`
	Type   string       `json:"type"`
	Files  []FileObject `json:"files"`
}

type FileObject struct {
	Type string     `json:"type"`
	Name string     `json:"name"`
	File NotionFile `json:"file"`
}

type UploadFileRequest struct {
	Bucket              string     `json:"bucket"`
	Name                string     `json:"name"`
	ContentType         string     `json:"contentType"`
	Record              RecordInfo `json:"record"`
	SupportExtraHeaders bool       `json:"supportExtraHeaders"`
	ContentLength       int64      `json:"contentLength"`
}

type RecordInfo struct {
	Table   string `json:"table"`
	ID      string `json:"id"`
	SpaceID string `json:"spaceId"`
}

type UploadResponse struct {
	Type                string       `json:"type"`
	URL                 string       `json:"url"`
	SignedGetUrl        string       `json:"signedGetUrl"`
	SignedUploadPostUrl string       `json:"signedUploadPostUrl"`
	SignedPutUrl        string       `json:"signedPutUrl"`
	PostHeaders         []string     `json:"postHeaders"`
	PutHeaders          []putHeader  `json:"putHeaders"`
	Fields              UploadFields `json:"fields"`
}

type putHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UploadFields struct {
	ContentType       string `json:"Content-Type"`
	XAmzStorageClass  string `json:"x-amz-storage-class"`
	Tagging           string `json:"tagging"`
	Bucket            string `json:"bucket"`
	XAmzAlgorithm     string `json:"X-Amz-Algorithm"`
	XAmzCredential    string `json:"X-Amz-Credential"`
	XAmzDate          string `json:"X-Amz-Date"`
	XAmzSecurityToken string `json:"X-Amz-Security-Token"`
	Key               string `json:"key"`
	Policy            string `json:"Policy"`
	XAmzSignature     string `json:"X-Amz-Signature"`
}

type UpdateFileStatusRequest struct {
	RequestID    string        `json:"requestId"`
	Transactions []Transaction `json:"transactions"`
}

type Transaction struct {
	ID      string      `json:"id"`
	SpaceID string      `json:"spaceId"`
	Debug   DebugInfo   `json:"debug"`
	Ops     []Operation `json:"operations"`
}

type DebugInfo struct {
	UserAction string `json:"userAction"`
}

type Operation struct {
	Pointer Pointer     `json:"pointer"`
	Path    []string    `json:"path"`
	Command string      `json:"command"`
	Args    interface{} `json:"args"`
}

type Pointer struct {
	ID      string `json:"id"`
	Table   string `json:"table"`
	SpaceID string `json:"spaceId"`
}

type CreatePageRequest struct {
	Parent     Parent     `json:"parent"`
	Properties Properties `json:"properties"`
}

type Parent struct {
	DatabaseID string `json:"database_id"`
}

type Properties struct {
	Title TitleProperty `json:"Title"`
}

type TitleProperty struct {
	Title []TitleText `json:"title"`
}

type RichTextProperty struct {
	RichText []RichText `json:"rich_text"`
}

type TitleText struct {
	Text TextContent `json:"text"`
}

type RichText struct {
	Text TextContent `json:"text"`
}

type TextContent struct {
	Content string `json:"content"`
}

type CreatePageResponse struct {
	ID         string     `json:"id"`
	Parent     Parent     `json:"parent"`
	Properties Properties `json:"properties"`
}

// ChunkFileStream 实现model.FileStreamer接口，用于分块上传
type ChunkFileStream struct {
	io.Reader
	name     string
	size     int64
	mimetype string
}

func (c *ChunkFileStream) GetName() string {
	return c.name
}

func (c *ChunkFileStream) GetSize() int64 {
	return c.size
}

func (c *ChunkFileStream) GetMimetype() string {
	return c.mimetype
}

func (c *ChunkFileStream) Close() error {
	if closer, ok := c.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (c *ChunkFileStream) GetID() string {
	return ""
}

func (c *ChunkFileStream) GetPath() string {
	return ""
}

func (c *ChunkFileStream) ModTime() time.Time {
	return time.Now()
}

func (c *ChunkFileStream) IsDir() bool {
	return false
}

func (c *ChunkFileStream) NeedStore() bool {
	return false
}

func (c *ChunkFileStream) IsForceStreamUpload() bool {
	return false
}

func (c *ChunkFileStream) GetExist() model.Obj {
	return nil
}

func (c *ChunkFileStream) CreateTime() time.Time {
	return time.Now()
}

func (c *ChunkFileStream) GetHash() utils.HashInfo {
	return utils.HashInfo{}
}

func (c *ChunkFileStream) SetExist(model.Obj) {
}

func (c *ChunkFileStream) RangeRead(httpRange http_range.Range) (io.Reader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *ChunkFileStream) CacheFullInTempFile() (model.File, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *ChunkFileStream) SetTmpFile(*os.File) {
}

func (c *ChunkFileStream) GetFile() model.File {
	return nil
}

// ChunkedRangeReadCloser 处理分块文件的Range请求
type ChunkedRangeReadCloser struct {
	notionClient *NotionService
	chunks       []FileChunk
	fileSize     int64
	utils.Closers
}

func NewChunkedRangeReadCloser(notionClient *NotionService, chunks []FileChunk, fileSize int64) *ChunkedRangeReadCloser {
	return &ChunkedRangeReadCloser{
		notionClient: notionClient,
		chunks:       chunks,
		fileSize:     fileSize,
		Closers:      utils.EmptyClosers(),
	}
}

func (c *ChunkedRangeReadCloser) RangeRead(ctx context.Context, httpRange http_range.Range) (io.ReadCloser, error) {
	if httpRange.Length == -1 {
		httpRange.Length = c.fileSize - httpRange.Start
	}

	// 找到需要的分块
	var neededChunks []FileChunk
	requestEnd := httpRange.Start + httpRange.Length

	for _, chunk := range c.chunks {
		// 检查分块是否与请求范围重叠
		if chunk.StartOffset < requestEnd && chunk.EndOffset > httpRange.Start {
			neededChunks = append(neededChunks, chunk)
		}
	}

	if len(neededChunks) == 0 {
		return nil, fmt.Errorf("no chunks found for range %d-%d", httpRange.Start, requestEnd-1)
	}

	return &ChunkedReader{
		notionClient: c.notionClient,
		chunks:       neededChunks,
		requestStart: httpRange.Start,
		requestEnd:   requestEnd,
		currentChunk: 0,
	}, nil
}

// ChunkedReader 实现跨分块的流式读取
type ChunkedReader struct {
	notionClient   *NotionService
	chunks         []FileChunk
	requestStart   int64
	requestEnd     int64
	currentChunk   int
	currentReader  io.ReadCloser
	currentOffset  int64
	totalRead      int64
}

func (r *ChunkedReader) Read(p []byte) (n int, err error) {
	if r.totalRead >= r.requestEnd-r.requestStart {
		return 0, io.EOF
	}

	// 如果当前没有reader或者当前reader已读完，切换到下一个分块
	if r.currentReader == nil {
		if r.currentChunk >= len(r.chunks) {
			return 0, io.EOF
		}

		chunk := r.chunks[r.currentChunk]

		// 获取分块的下载链接
		property, err := r.notionClient.GetPageProperty(chunk.NotionPageID, r.notionClient.filePageID)
		if err != nil {
			return 0, fmt.Errorf("获取分块%d下载链接失败: %v", r.currentChunk, err)
		}

		if len(property.Files) == 0 {
			return 0, fmt.Errorf("分块%d没有文件", r.currentChunk)
		}

		// 计算在当前分块中的读取范围
		chunkStart := max(r.requestStart, chunk.StartOffset) - chunk.StartOffset
		chunkEnd := min(r.requestEnd, chunk.EndOffset) - chunk.StartOffset

		// 创建HTTP请求获取分块数据
		reader, err := r.createChunkReader(property.Files[0].File.URL, chunkStart, chunkEnd-chunkStart)
		if err != nil {
			return 0, fmt.Errorf("创建分块%d读取器失败: %v", r.currentChunk, err)
		}

		r.currentReader = reader
		r.currentOffset = chunkStart
	}

	// 从当前reader读取数据
	remainingBytes := r.requestEnd - r.requestStart - r.totalRead
	if int64(len(p)) > remainingBytes {
		p = p[:remainingBytes]
	}

	n, err = r.currentReader.Read(p)
	r.totalRead += int64(n)

	if err == io.EOF {
		r.currentReader.Close()
		r.currentReader = nil
		r.currentChunk++

		// 如果还有更多数据要读取，继续下一个分块
		if r.totalRead < r.requestEnd-r.requestStart && r.currentChunk < len(r.chunks) {
			err = nil
		}
	}

	return n, err
}

func (r *ChunkedReader) Close() error {
	if r.currentReader != nil {
		return r.currentReader.Close()
	}
	return nil
}

func (r *ChunkedReader) createChunkReader(url string, offset, length int64) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置Range头
	if length > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %v", err)
	}

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
