package template

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Notion struct {
	model.Storage
	Addition
	db           *gorm.DB
	notionClient *NotionService
}

func (d *Notion) Config() driver.Config {
	return config
}

func (d *Notion) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *Notion) Init(ctx context.Context) error {
	// 初始化数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.DBUser, d.DBPass, d.DBHost, d.DBPort, d.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	// 自动迁移数据库表
	err = db.AutoMigrate(&Directory{}, &File{})
	if err != nil {
		return fmt.Errorf("迁移数据库失败: %v", err)
	}

	// 检查是否存在根目录，如果不存在则创建
	var rootDir Directory
	if err := db.Where("id = ?", 1).First(&rootDir).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建根目录
			rootDir = Directory{
				ID:        1,
				Name:      "/",
				ParentID:  nil,
				Deleted:   false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := db.Create(&rootDir).Error; err != nil {
				return fmt.Errorf("创建根目录失败: %v", err)
			}
		} else {
			return fmt.Errorf("检查根目录失败: %v", err)
		}
	}

	// 初始化Notion客户端
	d.notionClient = NewNotionService(d.NotionCookie, d.NotionToken, d.NotionSpaceID, d.NotionDatabaseID, d.NotionFilePageID)
	d.db = db

	return nil
}

func (d *Notion) Drop(ctx context.Context) error {
	return nil
}

func (d *Notion) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	var objs []model.Obj
	dirID := 1
	if dir != nil {
		id, _ := strconv.Atoi(dir.GetID())
		dirID = id
	}

	// 获取目录列表
	var directories []Directory
	if err := d.db.Where("parent_id = ? AND deleted = ?", dirID, false).Find(&directories).Error; err != nil {
		return nil, fmt.Errorf("获取目录列表失败: %v", err)
	}

	for _, dir := range directories {
		objs = append(objs, &model.Object{
			ID:       strconv.Itoa(dir.ID),
			Name:     dir.Name,
			Size:     0,
			Modified: dir.UpdatedAt,
			IsFolder: true,
		})
	}

	// 获取文件列表
	var files []File
	if err := d.db.Where("directory_id = ? AND deleted = ?", dirID, false).Find(&files).Error; err != nil {
		return nil, fmt.Errorf("获取文件列表失败: %v", err)
	}

	for _, file := range files {
		objs = append(objs, &model.Object{
			ID:       strconv.Itoa(file.ID),
			Name:     file.Name,
			Size:     file.Size,
			Modified: file.UpdatedAt,
			IsFolder: false,
		})
	}

	return objs, nil
}

func (d *Notion) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	var f File
	if err := d.db.Where("id = ? AND deleted = ?", file.GetID(), false).First(&f).Error; err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	property, err := d.notionClient.GetPageProperty(f.NotionPageID, d.NotionFilePageID)
	if err != nil {
		return nil, fmt.Errorf("获取文件URL失败: %v", err)
	}

	return &model.Link{
		URL: property.Files[0].File.URL,
	}, nil
}

func (d *Notion) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) (model.Obj, error) {
	parentID := 1
	if parentDir != nil {
		id, _ := strconv.Atoi(parentDir.GetID())
		parentID = id
	}

	dir := &Directory{
		Name:     dirName,
		ParentID: &parentID,
	}
	if err := d.db.Create(dir).Error; err != nil {
		return nil, fmt.Errorf("创建目录失败: %v", err)
	}

	return &model.Object{
		ID:       strconv.Itoa(dir.ID),
		Name:     dir.Name,
		Size:     0,
		Modified: dir.UpdatedAt,
		IsFolder: true,
	}, nil
}

func (d *Notion) Move(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	if srcObj.IsDir() {
		var dir Directory
		if err := d.db.Where("id = ? AND deleted = ?", srcObj.GetID(), false).First(&dir).Error; err != nil {
			return nil, fmt.Errorf("获取目录信息失败: %v", err)
		}

		parentID, _ := strconv.Atoi(dstDir.GetID())
		dir.ParentID = &parentID
		if err := d.db.Save(&dir).Error; err != nil {
			return nil, fmt.Errorf("移动目录失败: %v", err)
		}

		return &model.Object{
			ID:       strconv.Itoa(dir.ID),
			Name:     dir.Name,
			Size:     0,
			Modified: dir.UpdatedAt,
			IsFolder: true,
		}, nil
	} else {
		var file File
		if err := d.db.Where("id = ? AND deleted = ?", srcObj.GetID(), false).First(&file).Error; err != nil {
			return nil, fmt.Errorf("获取文件信息失败: %v", err)
		}

		dirID, _ := strconv.Atoi(dstDir.GetID())
		file.DirectoryID = dirID
		if err := d.db.Save(&file).Error; err != nil {
			return nil, fmt.Errorf("移动文件失败: %v", err)
		}

		return &model.Object{
			ID:       strconv.Itoa(file.ID),
			Name:     file.Name,
			Size:     file.Size,
			Modified: file.UpdatedAt,
			IsFolder: false,
		}, nil
	}
}

func (d *Notion) Rename(ctx context.Context, srcObj model.Obj, newName string) (model.Obj, error) {
	if srcObj.IsDir() {
		var dir Directory
		if err := d.db.Where("id = ? AND deleted = ?", srcObj.GetID(), false).First(&dir).Error; err != nil {
			return nil, fmt.Errorf("获取目录信息失败: %v", err)
		}

		dir.Name = newName
		if err := d.db.Save(&dir).Error; err != nil {
			return nil, fmt.Errorf("重命名目录失败: %v", err)
		}

		return &model.Object{
			ID:       strconv.Itoa(dir.ID),
			Name:     dir.Name,
			Size:     0,
			Modified: dir.UpdatedAt,
			IsFolder: true,
		}, nil
	} else {
		var file File
		if err := d.db.Where("id = ? AND deleted = ?", srcObj.GetID(), false).First(&file).Error; err != nil {
			return nil, fmt.Errorf("获取文件信息失败: %v", err)
		}

		file.Name = newName
		if err := d.db.Save(&file).Error; err != nil {
			return nil, fmt.Errorf("重命名文件失败: %v", err)
		}

		return &model.Object{
			ID:       strconv.Itoa(file.ID),
			Name:     file.Name,
			Size:     file.Size,
			Modified: file.UpdatedAt,
			IsFolder: false,
		}, nil
	}
}

func (d *Notion) Copy(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	if srcObj.IsDir() {
		// 复制目录
		var srcDir Directory
		if err := d.db.Where("id = ? AND deleted = ?", srcObj.GetID(), false).First(&srcDir).Error; err != nil {
			return nil, fmt.Errorf("获取源目录信息失败: %v", err)
		}

		// 创建新目录
		dstDirID, _ := strconv.Atoi(dstDir.GetID())
		newDir := &Directory{
			Name:     srcDir.Name,
			ParentID: &dstDirID,
		}
		if err := d.db.Create(newDir).Error; err != nil {
			return nil, fmt.Errorf("创建目标目录失败: %v", err)
		}

		// 复制目录下的所有文件
		var files []File
		if err := d.db.Where("directory_id = ? AND deleted = ?", srcDir.ID, false).Find(&files).Error; err != nil {
			return nil, fmt.Errorf("获取源目录文件列表失败: %v", err)
		}

		for _, file := range files {
			newFile := &File{
				Name:         file.Name,
				Size:         file.Size,
				SHA1:         file.SHA1,
				NotionPageID: file.NotionPageID,
				DirectoryID:  newDir.ID,
			}
			if err := d.db.Create(newFile).Error; err != nil {
				return nil, fmt.Errorf("复制文件失败: %v", err)
			}
		}

		// 递归复制子目录
		var subDirs []Directory
		if err := d.db.Where("parent_id = ? AND deleted = ?", srcDir.ID, false).Find(&subDirs).Error; err != nil {
			return nil, fmt.Errorf("获取子目录列表失败: %v", err)
		}

		// 递归复制子目录
		for _, subDir := range subDirs {
			// 创建子目录对象
			subDirObj := &model.Object{
				ID:       strconv.Itoa(subDir.ID),
				Name:     subDir.Name,
				Size:     0,
				Modified: subDir.UpdatedAt,
				IsFolder: true,
			}
			// 递归复制子目录
			if _, err := d.Copy(ctx, subDirObj, &model.Object{
				ID:       strconv.Itoa(newDir.ID),
				Name:     newDir.Name,
				Size:     0,
				Modified: newDir.UpdatedAt,
				IsFolder: true,
			}); err != nil {
				return nil, fmt.Errorf("复制子目录 %s 失败: %v", subDir.Name, err)
			}
		}

		return &model.Object{
			ID:       strconv.Itoa(newDir.ID),
			Name:     newDir.Name,
			Size:     0,
			Modified: newDir.UpdatedAt,
			IsFolder: true,
		}, nil
	} else {
		// 复制文件
		var srcFile File
		if err := d.db.Where("id = ? AND deleted = ?", srcObj.GetID(), false).First(&srcFile).Error; err != nil {
			return nil, fmt.Errorf("获取源文件信息失败: %v", err)
		}

		// 创建新文件记录
		dstDirID, _ := strconv.Atoi(dstDir.GetID())
		newFile := &File{
			Name:         srcFile.Name,
			Size:         srcFile.Size,
			SHA1:         srcFile.SHA1,
			NotionPageID: srcFile.NotionPageID,
			DirectoryID:  dstDirID,
		}
		if err := d.db.Create(newFile).Error; err != nil {
			return nil, fmt.Errorf("复制文件记录失败: %v", err)
		}

		return &model.Object{
			ID:       strconv.Itoa(newFile.ID),
			Name:     newFile.Name,
			Size:     newFile.Size,
			Modified: newFile.UpdatedAt,
			IsFolder: false,
		}, nil
	}
}

func (d *Notion) Remove(ctx context.Context, obj model.Obj) error {
	if obj.IsDir() {
		if err := d.db.Model(&Directory{}).Where("id = ?", obj.GetID()).Update("deleted", true).Error; err != nil {
			return fmt.Errorf("删除目录失败: %v", err)
		}
	} else {
		if err := d.db.Model(&File{}).Where("id = ?", obj.GetID()).Update("deleted", true).Error; err != nil {
			return fmt.Errorf("删除文件失败: %v", err)
		}
	}
	return nil
}

func (d *Notion) Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) (model.Obj, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", filepath.Base(file.GetName())+".*")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 写入文件内容
	if _, err := io.Copy(tempFile, file); err != nil {
		return nil, fmt.Errorf("写入文件内容失败: %v", err)
	}

	// 计算文件SHA1
	sha1, err := d.notionClient.CalculateFileSHA1(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("计算SHA1失败: %v", err)
	}

	// 检查数据库中是否已存在相同SHA1的文件
	var existingFile File
	if err := d.db.Where("sha1 = ? AND deleted = ?", sha1, false).First(&existingFile).Error; err == nil {
		// 如果文件已存在，直接创建新的文件记录，但使用已存在的NotionPageID
		dirID, _ := strconv.Atoi(dstDir.GetID())
		newFile := &File{
			Name:         filepath.Base(file.GetName()),
			Size:         file.GetSize(),
			SHA1:         sha1,
			NotionPageID: existingFile.NotionPageID,
			DirectoryID:  dirID,
		}
		if err := d.db.Create(newFile).Error; err != nil {
			return nil, fmt.Errorf("创建文件记录失败: %v", err)
		}

		return &model.Object{
			ID:       strconv.Itoa(newFile.ID),
			Name:     newFile.Name,
			Size:     newFile.Size,
			Modified: newFile.UpdatedAt,
			IsFolder: false,
		}, nil
	}

	// 创建Notion页面
	pageID, err := d.notionClient.CreateDatabasePage(filepath.Base(file.GetName()), sha1)
	if err != nil {
		return nil, fmt.Errorf("创建Notion页面失败: %v", err)
	}

	// 上传文件到Notion
	err = d.notionClient.UploadAndUpdateFilePut(tempFile.Name(), pageID)
	if err != nil {
		return nil, fmt.Errorf("上传文件到Notion失败: %v", err)
	}

	// 保存到数据库
	dirID, _ := strconv.Atoi(dstDir.GetID())
	f := &File{
		Name:         filepath.Base(file.GetName()),
		Size:         file.GetSize(),
		SHA1:         sha1,
		NotionPageID: pageID,
		DirectoryID:  dirID,
	}
	if err := d.db.Create(f).Error; err != nil {
		return nil, fmt.Errorf("保存文件信息失败: %v", err)
	}

	return &model.Object{
		ID:       strconv.Itoa(f.ID),
		Name:     f.Name,
		Size:     f.Size,
		Modified: f.UpdatedAt,
		IsFolder: false,
	}, nil
}

func (d *Notion) GetArchiveMeta(ctx context.Context, obj model.Obj, args model.ArchiveArgs) (model.ArchiveMeta, error) {
	return nil, errs.NotImplement
}

func (d *Notion) ListArchive(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) ([]model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Notion) Extract(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) (*model.Link, error) {
	return nil, errs.NotImplement
}

func (d *Notion) ArchiveDecompress(ctx context.Context, srcObj, dstDir model.Obj, args model.ArchiveDecompressArgs) ([]model.Obj, error) {
	return nil, errs.NotImplement
}

var _ driver.Driver = (*Notion)(nil)
