# MDFS - Markdown 渲染测试

这是一个专为 **MDFS** 测试 GitHub 风格 Markdown 渲染的样例文件 `readme.md`。

---

## 💡 GitHub Alerts 警告块测试

> [!NOTE]
> 这是一个 **Note** 提示框，用于展示补充背景或通用信息。

> [!TIP]
> 这是一个 **Tip** 建议框，用于提示最佳实践或技巧。

> [!IMPORTANT]
> 这是一个 **Important** 重要提示，用于说明关键注意点。

> [!WARNING]
> 这是一个 **Warning** 警告框，提醒用户操作需谨慎。

> [!CAUTION]
> 这是一个 **Caution** 危险警示，提醒可能存在风险的行为。

---

## ✅ 任务清单 (Task Lists)

- [x] GitHub 风格排版与样式主题 (`github-markdown-css`)
- [x] 语法高亮代码块 (`highlight.js`)
- [x] 相对路径图片与超链接自动重写
- [x] XSS 安全防范 (`dompurify`)
- [ ] 后续可选扩展（如 KaTeX 数学公式、Mermaid 图表等）

---

## 📊 GFM 表格测试

| 功能特性 | 支持引擎 | 安全过滤 | 还原度 |
| :--- | :---: | :---: | :--- |
| Markdown 渲染 | `markdown-it` | ✅ DOMPurify | 100% 对标 GitHub |
| 代码语法高亮 | `highlight.js` | ✅ 内置转义 | 丰富主流语言支持 |
| 警告提示块 | `markdown-it-github-alerts` | ✅ 保留 Octicon 图标 | GitHub 原生规范 |

---

## 💻 代码高亮测试

### Go 代码示例
```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, MDFS Markdown Preview!")
}
```

### TypeScript / JavaScript 示例
```typescript
interface FileEntry {
  name: string
  kind: 'file' | 'directory'
  size: number
}

const isMarkdown = (name: string): boolean => /\.md$/i.test(name)
console.log(isMarkdown('readme.md')) // true
```

### Bash 示例
```bash
# 启动 MDFS 开发服务器
make dev-api
```

---

## 🔗 相对路径与链接测试

- 相对文本文件：[查看当前目录的 welcome.txt](./welcome.txt)
- 相对子目录链接：[进入子文件夹 t1](./t1)
- 外部绝对链接：[Vue 官方网站](https://vuejs.org/)

---
*Generated for MDFS testing.*
