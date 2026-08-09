// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

const REPO = 'https://github.com/bmartel/alacris-go';

// GitHub Pages serves a project site from /<repo>/, so every asset and link has
// to be built with that prefix.
const BASE = '/alacris-go';

export default defineConfig({
  site: 'https://bmartel.github.io',
  base: BASE,

  integrations: [
    starlight({
      title: 'alacris-go',
      description:
        'Build alacris web components from Go and templ: typed wrappers generated from your define() calls, the runtime served from Go, and server-driven props with no HTML on the wire.',
      favicon: '/favicon.svg',
      social: [
        { icon: 'github', label: 'GitHub', href: REPO },
        { icon: 'seti:go', label: 'pkg.go.dev', href: 'https://pkg.go.dev/github.com/bmartel/alacris-go' },
      ],
      editLink: { baseUrl: `${REPO}/edit/main/docs/` },
      lastUpdated: true,
      customCss: ['./src/styles/docs.css'],

      head: [
        // Every live example on this site imports alacris under its real
        // specifier, which this map points at the bytes the Go module
        // vendors. What runs here is what your server would serve.
        {
          tag: 'script',
          attrs: { type: 'importmap' },
          content: JSON.stringify({
            imports: {
              alacris: `${BASE}/lib/alacris.js`,
              'alacris/store': `${BASE}/lib/store.js`,
              'alacris/context': `${BASE}/lib/context.js`,
              'alacris/signal': `${BASE}/lib/signal.js`,
            },
          }),
        },
        // The stand-in components the examples render. Loading them site-wide
        // keeps every page's markup live without each page wiring it up.
        { tag: 'script', attrs: { type: 'module', src: `${BASE}/demo-elements.js` } },
      ],

      sidebar: [
        {
          label: 'Start here',
          items: [
            { label: 'What is alacris-go?', slug: 'start/what-is-alacris-go' },
            { label: 'Installation', slug: 'start/installation' },
            { label: 'Your first component', slug: 'start/first-component' },
            { label: 'Upgrading', slug: 'start/upgrading' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'Rendering elements', slug: 'guides/rendering' },
            { label: 'Props and encoding', slug: 'guides/props' },
            { label: 'Generating wrappers', slug: 'guides/codegen' },
            { label: 'Theming', slug: 'guides/theming' },
            { label: 'Live: props from the server', slug: 'guides/live-props', badge: 'Live' },
            { label: 'Live: actions from the page', slug: 'guides/live-actions', badge: 'Live' },
            { label: 'Sessions and reconnects', slug: 'guides/sessions' },
            { label: 'Deploying', slug: 'guides/deploying' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Go API', slug: 'reference/api' },
            { label: 'CLI', slug: 'reference/cli' },
            { label: 'JSDoc tags', slug: 'reference/jsdoc' },
            { label: 'Wire protocol', slug: 'reference/wire-protocol' },
            { label: 'Security', slug: 'reference/security' },
            { label: 'Performance', slug: 'reference/performance' },
            { label: 'Limitations', slug: 'reference/limitations' },
            { label: 'AI agents', slug: 'reference/agents' },
          ],
        },
      ],
    }),
  ],
});
