# HTML 库 - 智能 HTML 内容提取

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Security](https://img.shields.io/badge/Security-Production%20Ready-green.svg)](SECURITY.md)

**生产级 Go 语言智能 HTML 内容提取库。** 100% 兼容 `golang.org/x/net/html` —— 可作为直接替代品使用，同时获得强大的内容提取功能。

#### **[📖 English Documentation](README.md)** - User guide

## ✨ 核心特性

### 🎯 智能内容提取
- **文章检测**：使用评分算法识别主要内容（文本密度、链接密度、语义标签）
- **智能文本提取**：保留结构、处理换行、计算字数和阅读时间
- **媒体提取**：提取图片、视频、音频及完整元数据（URL、尺寸、替代文本、类型检测）
- **链接分析**：外部/内部检测、nofollow 属性、锚文本提取

### 🚀 生产就绪的性能
- **内容寻址缓存**：基于 SHA256 的键值，支持 TTL 和 LRU 淘汰
- **批量处理**：可配置工作池的并行提取
- **线程安全**：无需外部同步即可并发使用
- **资源限制**：可配置输入大小、嵌套深度和超时保护

### 📦 零冗余

- **单一依赖**：仅依赖 `golang.org/x/net/html`（无臃肿的依赖树）
- **最小 API 接口**：简单、专注、易学（不是大杂烩）
- **无破坏性变更**：稳定的 API，保证向后兼容


### 🎯 使用场景
- 📰 **新闻聚合器**：从各种新闻网站提取干净的文章内容
- 🤖 **网页爬虫**：高效地从 HTML 页面获取结构化数据
- 📝 **内容管理**：将 HTML 转换为 Markdown 或其他格式
- 🔍 **搜索引擎**：索引主要内容，过滤导航/广告噪音
- 📊 **数据分析**：大规模提取和分析网页内容
- 📱 **RSS/Feed 生成器**：从 HTML 内容创建订阅源
- 🎓 **文档工具**：将 HTML 文档转换为其他格式


## 📥 安装

```bash
go get github.com/cybergodev/html
```

## 🚀 快速开始

### 智能内容提取

从复杂的 HTML 中提取干净、结构化的内容：

```go
import "github.com/cybergodev/html"

processor := html.NewWithDefaults()
defer processor.Close()

htmlContent := `
    <html>
    <body>
        <nav>跳过此导航</nav>
        <article>
            <h1>编写更好 Go 代码的 10 个技巧</h1>
            <p>Go 是一门强大的语言，强调简洁性...</p>
            <img src="diagram.png" alt="架构图" width="800">
            <p>关键原则包括...</p>
        </article>
        <aside>广告</aside>
    </body>
    </html>
`

result, err := processor.ExtractWithDefaults(htmlContent)
if err != nil {
    panic(err)
}

// 提取的内容（自动移除导航和广告）
fmt.Println("标题:", result.Title)           // "编写更好 Go 代码的 10 个技巧"
fmt.Println("文本:", result.Text)             // 仅包含干净的文章文本
fmt.Println("字数:", result.WordCount)        // 156
fmt.Println("阅读时间:", result.ReadingTime)  // 47s
fmt.Println("图片数:", len(result.Images))    // 1

// 图片元数据
for _, img := range result.Images {
    fmt.Printf("图片: %s (%s x %s)\n", img.URL, img.Width, img.Height)
    fmt.Printf("替代文本: %s\n", img.Alt)
}
```

## 🎯 核心功能

### 1. 智能文章检测

自动提取主要内容，同时移除噪音：

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 包含导航、广告、侧边栏的复杂页面
htmlContent := `
    <html>
    <nav>网站导航</nav>
    <aside>侧边栏广告</aside>
    <article>
        <h1>主要文章</h1>
        <p>这是用户真正想要阅读的内容...</p>
    </article>
    <footer>页脚链接</footer>
    </html>
`

config := html.ExtractConfig{
    ExtractArticle: true,  // 启用智能内容检测
}

result, _ := processor.Extract(htmlContent, config)
// result.Text 仅包含文章内容
// 导航、广告、侧边栏和页脚会自动移除
```

### 2. 丰富的媒体提取

提取所有媒体及完整元数据：

```go
result, _ := processor.ExtractWithDefaults(htmlContent)

// 带完整元数据的图片
for _, img := range result.Images {
    fmt.Printf("URL: %s\n", img.URL)
    fmt.Printf("替代文本: %s\n", img.Alt)
    fmt.Printf("尺寸: %s x %s\n", img.Width, img.Height)
    fmt.Printf("装饰性: %v\n", img.IsDecorative)
}

// 视频 URL
for _, video := range result.Videos {
    fmt.Printf("视频: %s (类型: %s)\n", video.URL, video.Type)
}

// 音频文件
for _, audio := range result.Audios {
    fmt.Printf("音频: %s (类型: %s)\n", audio.URL, audio.Type)
}

// 带分析的链接
for _, link := range result.Links {
    fmt.Printf("链接: %s -> %s\n", link.Text, link.URL)
    fmt.Printf("外部: %v, NoFollow: %v\n", link.IsExternal, link.IsNoFollow)
}
```

### 3. 内联图片格式化

控制图片在提取文本中的显示方式：

```go
htmlContent := `
    <article>
        <p>引言段落。</p>
        <img src="diagram.png" alt="系统架构">
        <p>如上图所示...</p>
    </article>
`

// Markdown 格式
config := html.ExtractConfig{
    InlineImageFormat: "markdown",
}
result, _ := processor.Extract(htmlContent, config)
// 输出: "引言段落。\n![系统架构](diagram.png)\n如上图所示..."

// HTML 格式
config.InlineImageFormat = "html"
result, _ = processor.Extract(htmlContent, config)
// 输出: "引言段落。\n<img src=\"diagram.png\" alt=\"系统架构\">\n如上图所示..."

// 占位符格式
config.InlineImageFormat = "placeholder"
result, _ = processor.Extract(htmlContent, config)
// 输出: "引言段落。\n[IMAGE:1]\n如上图所示..."
```

**格式选项：**
- `none`：从文本中移除图片（默认）
- `placeholder`：插入 `[IMAGE:1]`、`[IMAGE:2]` 等
- `markdown`：插入 `![alt](url)` 用于 Markdown 转换
- `html`：插入 `<img>` 标签用于 HTML 重建

### 4. 批量处理

使用工作池并行处理多个文档：

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 处理多个 HTML 字符串
htmlContents := []string{
    "<html><body><h1>页面 1</h1><p>内容 1</p></body></html>",
    "<html><body><h1>页面 2</h1><p>内容 2</p></body></html>",
    "<html><body><h1>页面 3</h1><p>内容 3</p></body></html>",
}

config := html.DefaultExtractConfig()
results, err := processor.ExtractBatch(htmlContents, config)

for i, result := range results {
    if result != nil {
        fmt.Printf("页面 %d: %s (%d 字)\n", i+1, result.Title, result.WordCount)
    }
}

// 或直接处理文件
filePaths := []string{"page1.html", "page2.html", "page3.html"}
results, err = processor.ExtractBatchFiles(filePaths, config)
```

### 5. 性能与缓存

内置缓存和监控：

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 提取内容（自动缓存）
result1, _ := processor.ExtractWithDefaults(htmlContent)

// 相同内容？立即命中缓存
result2, _ := processor.ExtractWithDefaults(htmlContent)

// 检查统计信息
stats := processor.GetStatistics()
fmt.Printf("总处理数: %d\n", stats.TotalProcessed)
fmt.Printf("缓存命中: %d (%.1f%%)\n", stats.CacheHits, 
    float64(stats.CacheHits)/float64(stats.TotalProcessed)*100)
fmt.Printf("平均时间: %v\n", stats.AverageProcessTime)
fmt.Printf("错误数: %d\n", stats.ErrorCount)

// 需要时清除缓存
processor.ClearCache()
```

**缓存特性：**
- 基于 SHA256 的内容寻址键（抗冲突）
- 基于 TTL 的过期机制（默认：1 小时）
- 缓存满时 LRU 淘汰
- 线程安全，最小锁竞争

## ⚙️ 配置

### 处理器配置

自定义资源限制和行为：

```go
config := html.Config{
    MaxInputSize:       50 * 1024 * 1024,  // 最大输入 50MB
    ProcessingTimeout:  30 * time.Second,   // 处理超时 30 秒
    MaxCacheEntries:    1000,               // 缓存最多 1000 个结果
    CacheTTL:           time.Hour,          // 缓存 TTL 1 小时
    WorkerPoolSize:     4,                  // 批量处理 4 个并行工作器
    EnableSanitization: true,               // 清理 HTML 输入
    MaxDepth:           100,                // 最大 HTML 嵌套深度
}

processor, err := html.New(config)
if err != nil {
    log.Fatal(err)
}
defer processor.Close()
```

**默认值**（通过 `html.NewWithDefaults()`）：
- MaxInputSize: 50MB
- ProcessingTimeout: 30 秒
- MaxCacheEntries: 1000
- CacheTTL: 1 小时
- WorkerPoolSize: 4
- EnableSanitization: true
- MaxDepth: 100

### 提取配置

控制提取内容和方式：

```go
config := html.ExtractConfig{
    ExtractArticle:    true,        // 启用智能文章检测
    PreserveImages:    true,        // 提取图片元数据
    PreserveLinks:     true,        // 提取链接元数据
    PreserveVideos:    true,        // 提取视频元数据
    PreserveAudios:    true,        // 提取音频元数据
    InlineImageFormat: "markdown",  // none, placeholder, markdown, html
}

result, err := processor.Extract(htmlContent, config)
```

**快速默认配置：**
```go
// 启用所有功能，无内联图片
config := html.DefaultExtractConfig()

// 或使用简写
result, _ := processor.ExtractWithDefaults(htmlContent)
```

## 📚 API 参考

### 处理器方法

```go
// 创建处理器
processor := html.NewWithDefaults()
processor, err := html.New(config)
defer processor.Close()

// 提取内容
result, err := processor.Extract(htmlContent, config)
result, err := processor.ExtractWithDefaults(htmlContent)
result, err := processor.ExtractFromFile("page.html", config)

// 批量处理
results, err := processor.ExtractBatch(htmlContents, config)
results, err := processor.ExtractBatchFiles(filePaths, config)

// 监控
stats := processor.GetStatistics()
processor.ClearCache()
```

### 结果结构

```go
type Result struct {
    Text           string        // 提取的干净文本
    Title          string        // 页面/文章标题
    Images         []ImageInfo   // 图片元数据
    Links          []LinkInfo    // 链接元数据
    Videos         []VideoInfo   // 视频元数据
    Audios         []AudioInfo   // 音频元数据
    ProcessingTime time.Duration // 处理时长
    WordCount      int           // 字数
    ReadingTime    time.Duration // 预估阅读时间（200 字/分钟）
}
```

### 媒体类型

```go
type ImageInfo struct {
    URL          string  // 图片 URL
    Alt          string  // 替代文本
    Title        string  // title 属性
    Width        string  // width 属性
    Height       string  // height 属性
    IsDecorative bool    // 如果替代文本为空则为 true
    Position     int     // 在文本中的位置（用于内联格式化）
}

type LinkInfo struct {
    URL        string  // 链接 URL
    Text       string  // 锚文本
    Title      string  // title 属性
    IsExternal bool    // 如果是外部域名则为 true
    IsNoFollow bool    // 如果有 rel="nofollow" 则为 true
}

type VideoInfo struct {
    URL      string  // 视频 URL（原生、YouTube、Vimeo、直接链接）
    Type     string  // MIME 类型或 "embed"
    Poster   string  // 海报图片 URL
    Width    string  // width 属性
    Height   string  // height 属性
    Duration string  // duration 属性
}

type AudioInfo struct {
    URL      string  // 音频 URL
    Type     string  // MIME 类型
    Duration string  // duration 属性
}
```

### 统计信息

```go
type Statistics struct {
    TotalProcessed     int64         // 执行的总提取次数
    CacheHits          int64         // 缓存命中次数
    CacheMisses        int64         // 缓存未命中次数
    ErrorCount         int64         // 总错误数
    AverageProcessTime time.Duration // 平均处理时间
}
```

## 💡 使用示例

查看 [examples/](examples) 目录获取完整的可运行示例：

- **[basic_extraction.go](examples/basic_extraction.go)** - 简单内容提取
- **[article_detection.go](examples/article_detection.go)** - 智能文章提取
- **[blog_post_extraction.go](examples/blog_post_extraction.go)** - 真实博客文章提取
- **[media_extraction.go](examples/media_extraction.go)** - 提取图片、视频、音频、链接
- **[inline_images.go](examples/inline_images.go)** - 图片格式化选项
- **[batch_processing.go](examples/batch_processing.go)** - 并行处理
- **[concurrent_usage.go](examples/concurrent_usage.go)** - 线程安全的并发使用
- **[caching_performance.go](examples/caching_performance.go)** - 缓存和性能
- **[custom_configuration.go](examples/custom_configuration.go)** - 自定义设置
- **[standard_html_parsing.go](examples/standard_html_parsing.go)** - 标准 HTML API

## 🔒 线程安全

`Processor` **可安全地被多个 goroutine 并发使用**，无需外部同步：

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 可以安全地从多个 goroutine 调用
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        result, _ := processor.ExtractWithDefaults(htmlContent)
        fmt.Printf("Goroutine %d: %s\n", id, result.Title)
    }(i)
}
wg.Wait()
```

## ⚡ 性能提示

1. **重用处理器**：创建一次，多次使用（避免每次请求都创建）
2. **启用缓存**：默认设置效果良好（1000 条目，1 小时 TTL）
3. **批量处理**：对多个文档使用 `ExtractBatch()`（并行工作器）
4. **调整限制**：根据内容调整 `MaxInputSize`（默认：50MB）
5. **工作池**：将 `WorkerPoolSize` 设置为匹配 CPU 核心数（默认：4）

## 🔄 与 golang.org/x/net/html 的兼容性

### 标准 HTML 解析（100% 兼容）

本库是 `golang.org/x/net/html` 的 **100% 兼容直接替代品**：

```go
// 之前
import "golang.org/x/net/html"

// 之后  
import "github.com/cybergodev/html"

// 解析 HTML 文档
doc, err := html.Parse(strings.NewReader(htmlContent))

// 渲染为 HTML
html.Render(os.Stdout, doc)

// 转义/反转义 HTML 实体
escaped := html.EscapeString("<script>alert('xss')</script>")
unescaped := html.UnescapeString("&lt;html&gt; &copy; 2024")

// HTML 标记化
tokenizer := html.NewTokenizer(strings.NewReader("<p>Test</p>"))
```

所有 `golang.org/x/net/html` API 的工作方式完全相同 —— 只需更改导入：

**重新导出的内容：**
- 所有类型：`Node`、`Token`、`Tokenizer`、`Attribute`、`NodeType`、`TokenType`
- 所有函数：`Parse()`、`ParseFragment()`、`Render()`、`EscapeString()`、`UnescapeString()`、`NewTokenizer()`
- 所有常量：`ElementNode`、`TextNode`、`DocumentNode`、`CommentNode`、`DoctypeNode` 等

**迁移成本：** 零。只需更改导入路径。

查看 [COMPATIBILITY.md](COMPATIBILITY.md) 获取详细的兼容性信息。


---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

## 🤝 贡献

欢迎贡献！请随时提交 Pull Request。对于重大更改，请先开启 issue 讨论您想要更改的内容。

## 🌟 Star 历史

如果您觉得这个项目有用，请考虑给它一个 star！⭐

---

**由 CyberGoDev 团队用 ❤️ 制作**
