# Notion 驱动大文件下载问题修复报告

## 问题分析

通过分析 Notion 驱动的代码，发现了以下导致无法下载大于 5GB 文件的问题：

### 1. 包名不一致
- 所有文件使用了 `template` 包名而不是 `notion`
- 这可能导致驱动注册失败

### 2. HTTP 客户端配置问题
- `createChunkReader` 方法中的 HTTP 客户端没有设置超时
- 缺少必要的 HTTP 头部设置
- 没有重试机制

### 3. 错误处理不完善
- 分块下载时缺少重试逻辑
- 错误信息不够详细，难以调试

## 修复内容

### 1. 修复包名
- 将所有文件的包名从 `template` 改为 `notion`

### 2. 改进 HTTP 客户端配置
```go
// 设置User-Agent和其他必要的头部
req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
req.Header.Set("Accept", "*/*")
req.Header.Set("Accept-Encoding", "identity")

// 创建带超时的HTTP客户端，适合大文件下载
client := &http.Client{
    Timeout: time.Minute * 30, // 30分钟超时，适合大文件分块下载
}
```

### 3. 添加重试机制
- 在 `ChunkedReader.Read` 方法中添加最多3次重试
- 使用递增延迟策略
- 改进错误信息，包含重试次数

### 4. 改进边界检查
- 在 `RangeRead` 方法中添加更严格的边界检查
- 确保请求范围不超过文件大小
- 提供更详细的错误信息

## 关键改进点

### 1. 超时设置
原来的 HTTP 客户端没有超时设置，可能导致大文件下载时连接超时。现在设置了30分钟超时，适合大文件分块下载。

### 2. 重试逻辑
```go
// 重试逻辑：最多重试3次
maxRetries := 3
for retry := 0; retry < maxRetries; retry++ {
    // 获取分块的下载链接
    property, err := r.notionClient.GetPageProperty(chunk.NotionPageID, r.notionClient.filePageID)
    if err != nil {
        if retry == maxRetries-1 {
            return 0, fmt.Errorf("获取分块%d下载链接失败(重试%d次): %v", r.currentChunk, retry+1, err)
        }
        time.Sleep(time.Second * time.Duration(retry+1)) // 递增延迟
        continue
    }
    // ... 成功处理
    break
}
```

### 3. 更好的错误处理
- 非EOF错误时关闭当前reader，下次读取时会重新创建
- 提供更详细的错误信息，包含文件大小、分块数量等上下文

## 测试建议

1. **小文件测试**：先测试小于5GB的文件，确保基本功能正常
2. **大文件测试**：测试5GB以上的文件，验证分块下载功能
3. **网络中断测试**：模拟网络中断，验证重试机制
4. **并发下载测试**：测试多个文件同时下载的情况

## 预期效果

修复后，Notion 驱动应该能够：
1. 正常下载大于5GB的文件
2. 在网络不稳定时自动重试
3. 提供更好的错误信息便于调试
4. 支持断点续传（通过Range请求）

## 注意事项

1. 确保 Notion API 的文件URL没有过期
2. 监控下载过程中的内存使用情况
3. 大文件下载可能需要较长时间，建议设置合适的超时时间
