declare module 'markdown-it-task-lists' {
  import type MarkdownIt from 'markdown-it'
  const taskLists: MarkdownIt.PluginSimple
  export default taskLists
}

declare module 'markdown-it-github-alerts' {
  import type MarkdownIt from 'markdown-it'
  const alerts: MarkdownIt.PluginSimple
  export default alerts
}
