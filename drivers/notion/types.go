package template

import (
	"time"
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
