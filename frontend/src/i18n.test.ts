import { describe, expect, it } from 'vitest'
import { translate } from './i18n'

describe('translations', () => {
  it('provides all supported language labels', () => {
    expect(translate('zh-CN', 'uploadFiles')).toBe('上传文件')
    expect(translate('zh-TW', 'uploadFiles')).toBe('上傳檔案')
    expect(translate('en', 'uploadFiles')).toBe('Upload files')
  })
})
