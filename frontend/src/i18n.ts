import type { Language } from './types/files'

const messages = {
  'zh-CN': {
    uploadFiles: '上传文件', uploadFolder: '上传目录', newText: '新建文本', newFolder: '新建目录', login: '登录', logout: '退出登录',
    localSearch: '在当前页面中搜索', globalSearch: '在整个服务器中搜索', root: '根目录', back: '返回上一层', file: '文件', size: '大小', modified: '修改时间', actions: '操作',
    download: '下载', archive: '打包下载', qr: '扫码下载', rename: '重命名', edit: '编辑文本', remove: '删除', preview: '预览', play: '播放',
    username: '用户', password: '密码', cancel: '取消', close: '关闭', create: '创建', save: '保存', oldName: '旧名字', newName: '新名字', title: '标题', content: '内容',
    deleteQuestion: '确定删除', copyLink: '复制链接', copied: '已复制', empty: '目录为空', loading: '正在读取目录…', searchResults: '搜索结果', tooMany: '搜索结果已截断',
    uploadProgress: '上传进度', uploadDone: '上传完成', uploadCanceled: '上传已取消', error: '操作失败', languageName: '简体中文', confirm: '确定', dragHere: '释放鼠标以上传文件',
  },
  'zh-TW': {
    uploadFiles: '上傳檔案', uploadFolder: '上傳目錄', newText: '新增文字', newFolder: '新增目錄', login: '登入', logout: '登出',
    localSearch: '在目前頁面中搜尋', globalSearch: '在整個伺服器中搜尋', root: '根目錄', back: '返回上一層', file: '檔案', size: '大小', modified: '修改時間', actions: '操作',
    download: '下載', archive: '打包下載', qr: '掃碼下載', rename: '重新命名', edit: '編輯文字', remove: '刪除', preview: '預覽', play: '播放',
    username: '使用者', password: '密碼', cancel: '取消', close: '關閉', create: '建立', save: '儲存', oldName: '舊名稱', newName: '新名稱', title: '標題', content: '內容',
    deleteQuestion: '確定刪除', copyLink: '複製連結', copied: '已複製', empty: '目錄為空', loading: '正在讀取目錄…', searchResults: '搜尋結果', tooMany: '搜尋結果已截斷',
    uploadProgress: '上傳進度', uploadDone: '上傳完成', uploadCanceled: '上傳已取消', error: '操作失敗', languageName: '繁體中文', confirm: '確定', dragHere: '放開滑鼠以上傳檔案',
  },
  en: {
    uploadFiles: 'Upload files', uploadFolder: 'Upload folder', newText: 'New text', newFolder: 'New folder', login: 'Login', logout: 'Logout',
    localSearch: 'Search in this page', globalSearch: 'Search entire server', root: 'Root', back: 'Up one level', file: 'File', size: 'Size', modified: 'Modified', actions: 'Actions',
    download: 'Download', archive: 'Download ZIP', qr: 'QR download', rename: 'Rename', edit: 'Edit text', remove: 'Delete', preview: 'Preview', play: 'Play',
    username: 'Username', password: 'Password', cancel: 'Cancel', close: 'Close', create: 'Create', save: 'Save', oldName: 'Old name', newName: 'New name', title: 'Title', content: 'Content',
    deleteQuestion: 'Delete', copyLink: 'Copy link', copied: 'Copied', empty: 'This folder is empty', loading: 'Loading directory…', searchResults: 'Search results', tooMany: 'Search results were truncated',
    uploadProgress: 'Upload progress', uploadDone: 'Upload complete', uploadCanceled: 'Upload canceled', error: 'Operation failed', languageName: 'English', confirm: 'Confirm', dragHere: 'Drop files to upload',
  },
} as const

export type MessageKey = keyof typeof messages['zh-CN']
export const languages: Language[] = ['zh-CN', 'zh-TW', 'en']
export const translate = (language: Language, key: MessageKey) => messages[language][key]
