# HTML 库 - 智能 HTML 内容提取

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Security](https://img.shields.io/badge/Security-Production%20Ready-green.svg)](SECURITY.md)

**生产级 Go 库，用于智能 HTML 内容提取。** 100% 兼容 `golang.org/x/net/html` — 可作为直接替换使用，同时获得强大的内容提取功能。

#### **[📖 English Documentation](README.md)** - 用户指南

## ✨ 核心特性

### 🎯 智能内容提取
- **文章检测**: 使用评分算法识别主要内容（文本密度、链接密度、语义标签）
- **智能文本提取**: 保留结构，处理换行，计算字数和阅读时间
- **媒体提取**: 图片、视频、音频及完整元数据（URL、尺寸、alt 文本、类型检测）
- **链接分析**: 外部/内部检测、nofollow 属性、锚文本提取

### 🚀 生产就绪的性能
- **内容可寻址缓存**: 基于 SHA256 的键值，支持 TTL 和 LRU 淘汰
- **批量处理**: 并行提取，可配置工作池
- **线程安全**: 无需外部同步即可并发使用
- **资源限制**: 可配置输入大小、嵌套深度和超时保护

### 📦 零冗余

- **单一依赖**: 仅依赖 `golang.org/x/net/html`（无臃肿的依赖树）
- **最小 API 表面**: 简单、专注、易学

### 🎯 使用场景
- 📰 **新闻聚合器**: 从各种新闻网站提取干净的文章内容
- 🤖 **网页爬虫**: 高效地从 HTML 页面获取结构化数据
- 📝 **内容管理**: 将 HTML 转换为 Markdown 或其他格式
- 🔍 **搜索引擎**: 索引主要内容，过滤导航/广告噪音
- 📊 **数据分析**: 大规模提取和分析网页内容
- 📱 **RSS/Feed 生成器**: 从 HTML 内容创建 feed
- 🎓 **文档工具**: 将 HTML 文档转换为其他格式


## 📥 安装

```bash
go get github.com/cybergodev/html
```

## 🚀 快速开始

### 智能内容提取

从复杂 HTML 中提取干净、结构化的内容：

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
            <p>Go 是一门强调简洁性的强大语言...</p>
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
fmt.Println("标题:", result.Title)            // "编写更好 Go 代码的 10 个技巧"
fmt.Println("文本:", result.Text)             // 仅包含干净的文章文本
fmt.Println("字数:", result.WordCount)        // 8
fmt.Println("阅读时间:", result.ReadingTime)   // 2.4s
fmt.Println("图片数量:", len(result.Images))   // 1

// 图片元数据
for _, img := range result.Images {
    fmt.Printf("图片: %s (%s x %s)\n", img.URL, img.Width, img.Height)
    fmt.Printf("Alt: %s\n", img.Alt)
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
        <p>这是用户真正想要阅读的实际内容...</p>
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
    fmt.Printf("Alt: %s\n", img.Alt)
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
        <p>介绍段落。</p>
        <img src="diagram.png" alt="系统架构">
        <p>如上图所示...</p>
    </article>
`

// Markdown 格式
config := html.ExtractConfig{
    InlineImageFormat: "markdown",
}
result, _ := processor.Extract(htmlContent, config)
// 输出: "介绍段落。\n![系统架构](diagram.png)\n如上图所示..."

// HTML 格式
config.InlineImageFormat = "html"
result, _ = processor.Extract(htmlContent, config)
// 输出: "介绍段落。\n<img src=\"diagram.png\" alt=\"系统架构\">\n如上图所示..."

// 占位符格式
config.InlineImageFormat = "placeholder"
result, _ = processor.Extract(htmlContent, config)
// 输出: "介绍段落。\n[IMAGE:1]\n如上图所示..."
```

**格式选项:**
- `none`: 从文本中移除图片（默认）
- `placeholder`: 插入 `[IMAGE:1]`, `[IMAGE:2]` 等
- `markdown`: 插入 `![alt](url)` 用于 Markdown 转换
- `html`: 插入 `<img>` 标签用于 HTML 重构

### 4. 全面的链接提取

提取所有类型的资源链接，自动解析 URL：

```go
htmlContent := `
    <!DOCTYPE html>
    <html>
    <head>
        <base href="https://example.com/">
        <link rel="stylesheet" href="css/main.css">
        <script src="js/app.js"></script>
        <link rel="icon" href="/favicon.ico">
    </head>
    <body>
        <a href="/about">关于</a>
        <a href="https://external.com">外部链接</a>
        <img src="images/hero.jpg" alt="主图">
        <video src="videos/demo.mp4"></video>
        <audio src="audio/music.mp3"></audio>
        <iframe src="https://youtube.com/embed/abc123"></iframe>
    </body>
    </html>
`

// 简单提取（便利函数）
links, err := html.ExtractAllLinks(htmlContent)
if err != nil {
    log.Fatal(err)
}

// 使用便利函数按类型分组链接
linksByType := html.GroupLinksByType(links)

// 直接访问特定类型的数据
cssLinks := linksByType["css"]
jsLinks := linksByType["js"]
contentLinks := linksByType["link"]
images := linksByType["image"]
```

**高级配置:**
```go
processor := html.NewWithDefaults()
defer processor.Close()

config := html.LinkExtractionConfig{
    ResolveRelativeURLs:  true,  // 自动解析相对 URL
    BaseURL:              "",    // 自动检测或指定基础 URL
    IncludeImages:        true,  // 提取图片资源
    IncludeVideos:        true,  // 提取视频资源  
    IncludeAudios:        true,  // 提取音频资源
    IncludeCSS:           true,  // 提取 CSS 样式表
    IncludeJS:            true,  // 提取 JavaScript 文件
    IncludeContentLinks:  true,  // 提取导航链接
    IncludeExternalLinks: true,  // 提取外部域名链接
    IncludeIcons:         true,  // 提取图标和 favicon
}

links, err := processor.ExtractAllLinks(htmlContent, config)
```

**功能特性:**
- **自动 URL 解析**: 从 `<base>` 标签、canonical meta 或现有 URL 检测基础 URL
- **资源类型检测**: 图片、视频、音频、CSS、JS、内容链接、外部链接、图标
- **智能去重**: 防止结果中出现重复链接
- **域名分类**: 区分内部与外部链接
- **全面覆盖**: 从所有 HTML 元素提取，包括 `<link>`, `<script>`, `<img>`, `<video>`, `<audio>`, `<iframe>`, `<embed>`, `<object>`

### 5. 批量处理

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

### 6. 性能与缓存

内置缓存和监控：

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 提取内容（自动缓存）
result1, _ := processor.ExtractWithDefaults(htmlContent)

// 相同内容？立即缓存命中
result2, _ := processor.ExtractWithDefaults(htmlContent)

// 检查统计信息
stats := processor.GetStatistics()
fmt.Printf("总处理数: %d\n", stats.TotalProcessed)
fmt.Printf("缓存命中: %d (%.1f%%)\n", stats.CacheHits, 
    float64(stats.CacheHits)/float64(stats.TotalProcessed)*100)
fmt.Printf("平均时间: %v\n", stats.AverageProcessTime)
fmt.Printf("错误数: %d\n", stats.ErrorCount)

// 如需要可清除缓存
processor.ClearCache()
```

**缓存特性:**
- 基于 SHA256 的内容可寻址键（抗冲突）
- 基于 TTL 的过期（默认：1 小时）
- 缓存满时 LRU 淘汰
- 线程安全，最小锁竞争

## ⚙️ 配置

### 处理器配置

自定义资源限制和行为：

```go
config := html.Config{
    MaxInputSize:       50 * 1024 * 1024,   // 50MB 最大输入大小
    ProcessingTimeout:  30 * time.Second,   // 30s 处理超时
    MaxCacheEntries:    1000,               // 缓存最多 1000 个结果
    CacheTTL:           time.Hour,          // 1 小时缓存 TTL
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

**默认值**（通过 `html.NewWithDefaults()`）:
- MaxInputSize: 50MB
- ProcessingTimeout: 30s
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

**快速默认值:**
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
    ReadingTime    time.Duration // 预估阅读时间（200 WPM）
}
```

### 媒体类型

```go
type ImageInfo struct {
    URL          string  // 图片 URL
    Alt          string  // Alt 文本
    Title        string  // Title 属性
    Width        string  // Width 属性
    Height       string  // Height 属性
    IsDecorative bool    // Alt 文本为空时为 true
    Position     int     // 在文本中的位置（用于内联格式化）
}

type LinkInfo struct {
    URL        string  // 链接 URL
    Text       string  // 锚文本
    Title      string  // Title 属性
    IsExternal bool    // 外部域名时为 true
    IsNoFollow bool    // rel="nofollow" 时为 true
}

type VideoInfo struct {
    URL      string  // 视频 URL（原生、YouTube、Vimeo、直接）
    Type     string  // MIME 类型或 "embed"
    Poster   string  // 海报图片 URL
    Width    string  // Width 属性
    Height   string  // Height 属性
    Duration string  // Duration 属性
}

type AudioInfo struct {
    URL      string  // 音频 URL
    Type     string  // MIME 类型
    Duration string  // Duration 属性
}
```

### 统计信息

```go
type Statistics struct {
    TotalProcessed     int64         // 总提取次数
    CacheHits          int64         // 缓存命中
    CacheMisses        int64         // 缓存未命中
    ErrorCount         int64         // 总错误数
    AverageProcessTime time.Duration // 平均处理时间
}
```

## 💡 使用示例

查看 [examples/](examples) 目录获取完整的可运行示例：

- **[01_quick_start.go](examples/01_quick_start.go)** - 快速开始与便利函数
- **[02_content_extraction.go](examples/02_content_extraction.go)** - 内容提取与文章检测和内联图片
- **[03_link_extraction.go](examples/03_link_extraction.go)** - 全面链接提取与 URL 解析
- **[04_media_extraction.go](examples/04_media_extraction.go)** - 提取图片、视频、音频和链接及元数据
- **[05_advanced_usage.go](examples/05_advanced_usage.go)** - 高级功能：自定义配置、批量处理、缓存、并发
- **[06_compatibility.go](examples/06_compatibility.go)** - 100% 兼容 golang.org/x/net/html

## 🔒 线程安全

`Processor` **对多个 goroutine 并发使用是安全的**，无需外部同步：

```go
processor := html.NewWithDefaults()
defer processor.Close()

// 可安全地从多个 goroutine 调用
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

1. **重用处理器**: 创建一次，多次使用（避免每次请求都创建）
2. **启用缓存**: 默认设置效果良好（1000 条目，1 小时 TTL）
3. **批量处理**: 对多个文档使用 `ExtractBatch()`（并行工作器）
4. **调整限制**: 根据内容调整 `MaxInputSize`（默认：50MB）
5. **工作池**: 设置 `WorkerPoolSize` 匹配 CPU 核心数（默认：4）

## 🔄 与 golang.org/x/net/html 的兼容性

### 标准 HTML 解析（100% 兼容）

此库是 `golang.org/x/net/html` 的 **100% 兼容直接替换**：

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

// 标记化 HTML
tokenizer := html.NewTokenizer(strings.NewReader("<p>Test</p>"))
```

所有 `golang.org/x/net/html` API 都完全相同 — 只需更改导入：

**重新导出的内容:**
- 所有类型: `Node`, `Token`, `Tokenizer`, `Attribute`, `NodeType`, `TokenType`
- 所有函数: `Parse()`, `ParseFragment()`, `Render()`, `EscapeString()`, `UnescapeString()`, `NewTokenizer()`
- 所有常量: `ElementNode`, `TextNode`, `DocumentNode`, `CommentNode`, `DoctypeNode` 等

**迁移成本:** 零。只需更改导入路径。

查看 [COMPATIBILITY.md](COMPATIBILITY.md) 获取详细兼容性信息。


---

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

## 🤝 贡献

欢迎贡献！请随时提交 Pull Request。对于重大更改，请先开启 issue 讨论您想要更改的内容。

## 🌟 Star 历史

如果您觉得这个项目有用，请考虑给它一个 star！⭐

---

**由 CyberGoDev 团队用 ❤️ 制作**