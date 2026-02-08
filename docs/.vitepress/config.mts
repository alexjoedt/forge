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
  title: "Forge",
  description: "A CLI for automated git version tagging and changelog generation",
  head: [
    ['link', { rel: 'icon', href: '/assets/logo.png' }]
  ],
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    search: {
      provider: 'local'
    },
    logo: '/assets/logo.png',
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Reference', link: '/reference/configuration' },
      {
        text: 'Examples',
        items: [
          { text: 'Single App', link: '/examples/single-app' },
          { text: 'Monorepo', link: '/examples/monorepo' },
          { text: 'CI/CD Workflows', link: '/examples/ci-cd' },
        ]
      }
    ],
    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          items: [
            { text: 'Getting Started', link: '/guide/getting-started' },
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Quick Start', link: '/guide/quick-start' },
          ]
        },
        {
          text: 'Core Concepts',
          items: [
            { text: 'Version Schemes', link: '/guide/version-schemes' },
            { text: 'Bump Command', link: '/guide/bump' },
            { text: 'Interactive Mode', link: '/guide/interactive-mode' },
          ]
        },
        {
          text: 'Workflows',
          items: [
            { text: 'Hotfix Workflow', link: '/guide/hotfix' },
            { text: 'Monorepo Setup', link: '/guide/monorepo' },
            { text: 'Changelog Generation', link: '/guide/changelog' },
          ]
        },
        {
          text: 'Build & Deploy',
          items: [
            { text: 'Building Binaries', link: '/guide/build' },
            { text: 'Docker Images', link: '/guide/docker' },
            { text: 'Node.js Integration', link: '/guide/nodejs' },
          ]
        },
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'Configuration', link: '/reference/configuration' },
            { text: 'CLI Commands', link: '/reference/cli-commands' },
            { text: 'Template Variables', link: '/reference/template-variables' },
          ]
        },
      ],
      '/examples/': [
        {
          text: 'Examples',
          items: [
            { text: 'Single App', link: '/examples/single-app' },
            { text: 'Monorepo', link: '/examples/monorepo' },
            { text: 'CI/CD Workflows', link: '/examples/ci-cd' },
          ]
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/alexjoedt/forge' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 alexjoedt'
    },

    editLink: {
      pattern: 'https://github.com/alexjoedt/forge/edit/master/docs/:path',
      text: 'Edit this page on GitHub'
    }
  }
})
