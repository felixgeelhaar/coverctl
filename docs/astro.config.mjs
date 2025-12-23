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
      ],
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Introduction', link: '/' },
            { label: 'Installation', link: '/installation/' },
            { label: 'Quick Start', link: '/quick-start/' },
          ],
        },
        {
          label: 'CLI Reference',
          items: [
            { label: 'Overview', link: '/cli/' },
            { label: 'check', link: '/cli/check/' },
            { label: 'run', link: '/cli/run/' },
            { label: 'watch', link: '/cli/watch/' },
            { label: 'init', link: '/cli/init/' },
            { label: 'report', link: '/cli/report/' },
            { label: 'Other Commands', link: '/cli/other/' },
          ],
        },
        {
          label: 'Configuration',
          items: [
            { label: 'Config File', link: '/configuration/' },
            { label: 'Domains', link: '/configuration/domains/' },
            { label: 'Policies', link: '/configuration/policies/' },
            { label: 'Advanced', link: '/configuration/advanced/' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'CI Integration', link: '/guides/ci-integration/' },
            { label: 'Build Flags', link: '/guides/build-flags/' },
          ],
        },
        {
          label: 'Architecture',
          items: [
            { label: 'Overview', link: '/architecture/' },
            { label: 'Contributing', link: '/architecture/contributing/' },
          ],
        },
      ],
    }),
  ],
});
