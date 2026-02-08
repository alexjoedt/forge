---
layout: home

hero:
  name: "Forge"
  text: "Version Management for Go Projects"
  tagline: "A CLI for automated git version tagging and changelog generation. Supports SemVer, CalVer, and monorepo workflows."
  image:
    src: /assets/logo.png
    alt: Forge Logo
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/alexjoedt/forge

features:
  - icon: 🏷️
    title: Semantic & Calendar Versioning
    details: Full support for SemVer (v1.2.3) and CalVer with ISO week numbers (2025.44). Interactive bump prompts with version previews.
  - icon: 🔧
    title: Hotfix Workflow
    details: Create hotfix branches from any released tag, apply patches, and create sequenced hotfix tags — all without touching main.
  - icon: 📦
    title: Monorepo Support
    details: Independent versioning per application with namespaced tags. Build, tag, and release each app separately.
  - icon: 📋
    title: Changelog Generation
    details: Parse Conventional Commits to generate changelogs in Markdown, JSON, or plain text. Supports breaking change detection.
  - icon: 🐳
    title: Docker & Build Integration
    details: Cross-compile Go binaries and build Docker images for multiple platforms and registries with template-based tag naming.
  - icon: 🤖
    title: CI/CD & Scripting Ready
    details: JSON output mode, dry-run support, and non-interactive flags make Forge perfect for automated pipelines.
---

