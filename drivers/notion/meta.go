package template

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

type Addition struct {
	driver.RootID
	NotionCookie     string `json:"notion_cookie" required:"true"`
	NotionToken      string `json:"notion_token" required:"true"`
	NotionSpaceID    string `json:"notion_space_id" required:"true"`
	NotionDatabaseID string `json:"notion_database_id" required:"true"`
	DBUser           string `json:"db_user" default:"root"`
	DBPass           string `json:"db_pass" default:"123456"`
	DBHost           string `json:"db_host" default:"localhost"`
	DBPort           string `json:"db_port" default:"3306"`
	DBName           string `json:"db_name" default:"filesystem"`
}

var config = driver.Config{
	Name:              "Notion",
	LocalSort:         false,
	OnlyLocal:         false,
	OnlyProxy:         false,
	NoCache:           false,
	NoUpload:          false,
	NeedMs:            false,
	DefaultRoot:       "1",
	CheckStatus:       false,
	Alert:             "",
	NoOverwriteUpload: false,
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Notion{}
	})
}
