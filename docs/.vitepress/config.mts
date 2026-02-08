import { defineConfig } from 'vitepress'
import { mermaidPlugin } from './plugins/vitepress-mermaid'
import footnote from 'markdown-it-footnote'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  markdown: {
    config: (md) => {
      md.use(mermaidPlugin),
        md.use(footnote)
    },
  },
  title: "Documentation",
  description: "TODO",
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    search: {
      provider: 'local'
    },
    logo: '/assets/logo.png',
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Documentation', link: 'TODO' }
    ],
    sidebar: [
      {
        text: 'Examples',
        items: [
          { text: 'Markdown Examples', link: '/markdown-examples' },
          { text: 'Runtime API Examples', link: '/api-examples' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/alexjoedt/forge' }
    ]
  }
})
