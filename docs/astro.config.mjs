// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
  site: 'https://felixgeelhaar.github.io',
  base: '/coverctl',
  integrations: [
    starlight({
      title: 'coverctl',
      description: 'Domain-aware test coverage enforcement for Go teams',
      social: {
        github: 'https://github.com/felixgeelhaar/coverctl',
      },
      editLink: {
        baseUrl: 'https://github.com/felixgeelhaar/coverctl/edit/main/docs/',
      },
      customCss: ['./src/styles/custom.css'],
      head: [
        {
          tag: 'meta',
          attrs: {
            property: 'og:image',
            content: 'https://felixgeelhaar.github.io/coverctl/og-image.png',
          },
        },
        {
          tag: 'link',
          attrs: {
            rel: 'preconnect',
            href: 'https://fonts.googleapis.com',
          },
        },
        {
          tag: 'link',
          attrs: {
            rel: 'preconnect',
            href: 'https://fonts.gstatic.com',
            crossorigin: 'anonymous',
          },
        },
      ],
      expressiveCode: {
        themes: ['github-dark', 'github-light'],
        styleOverrides: {
          borderRadius: '10px',
          codeFontFamily: "'JetBrains Mono', 'Fira Code', monospace",
        },
      },
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Introduction', slug: '' },
            { label: 'Installation', slug: 'installation' },
            { label: 'Quick Start', slug: 'quick-start' },
          ],
        },
        {
          label: 'CLI Reference',
          items: [
            { label: 'Overview', slug: 'cli' },
            { label: 'check', slug: 'cli/check' },
            { label: 'run', slug: 'cli/run' },
            { label: 'watch', slug: 'cli/watch' },
            { label: 'init', slug: 'cli/init' },
            { label: 'report', slug: 'cli/report' },
            { label: 'Other Commands', slug: 'cli/other' },
          ],
        },
        {
          label: 'Configuration',
          items: [
            { label: 'Config File', slug: 'configuration' },
            { label: 'Domains', slug: 'configuration/domains' },
            { label: 'Policies', slug: 'configuration/policies' },
            { label: 'Advanced', slug: 'configuration/advanced' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'CI Integration', slug: 'guides/ci-integration' },
            { label: 'Build Flags', slug: 'guides/build-flags' },
          ],
        },
        {
          label: 'Architecture',
          items: [
            { label: 'Overview', slug: 'architecture' },
            { label: 'Contributing', slug: 'architecture/contributing' },
          ],
        },
      ],
    }),
  ],
});
