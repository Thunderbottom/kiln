// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  integrations: [
      starlight({
          title: 'kiln',
          routeMiddleware: './src/routeData.ts',
          logo: {
              light: './src/assets/logo.svg',
              dark: './src/assets/logo-dark.svg',
              replacesTitle: true,
          },
          social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/thunderbottom/kiln' }],
          sidebar: [
              {
                  label: 'Getting Started',
                  items: [
                      { label: 'Introduction', slug: 'introduction' },
                      { label: 'Installation', slug: 'installation' },
                      { label: 'Quick Start', slug: 'quick-start' },
                      { label: 'Basic Concepts', slug: 'basic-concepts' },
                  ],
              },
              {
                  label: 'Configuration',
                  items: [
                      { label: 'Configuration File', slug: 'configuration/configuration-file' },
                      { label: 'Recipients and Groups', slug: 'configuration/recipients' },
                      { label: 'File Access Control', slug: 'configuration/access-control' },
                      { label: 'Environment Variables', slug: 'configuration/environment-variables' },
                  ],
              },
              {
                  label: 'Commands',
                  items: [
                      { label: 'Overview', slug: 'commands/overview' },
                      { label: 'init', slug: 'commands/init' },
                      { label: 'set', slug: 'commands/set' },
                      { label: 'get', slug: 'commands/get' },
                      { label: 'edit', slug: 'commands/edit' },
                      { label: 'export', slug: 'commands/export' },
                      { label: 'apply', slug: 'commands/apply' },
                      { label: 'run', slug: 'commands/run' },
                      { label: 'rekey', slug: 'commands/rekey' },
                      { label: 'info', slug: 'commands/info' },
                  ],
              },
              {
                  label: 'Example Workflows',
                  items: [
                      { label: 'Team Setup', slug: 'workflows/team-setup' },
                      { label: 'Adding Members', slug: 'workflows/adding-members' },
                      { label: 'Access Management', slug: 'workflows/access-management' },
                  ],
              },
              {
                  label: 'Reference',
                  items: [
                      { label: 'Configuration Schema', slug: 'reference/configuration' },
                      { label: 'Command Reference', slug: 'reference/commands' },
                      { label: 'Environment Variables', slug: 'reference/env-vars' },
                      { label: 'File Formats', slug: 'reference/formats' },
                      { label: 'Exit Codes', slug: 'reference/exit-codes' },
                  ],
              },
              { label: 'Go Library', slug: 'library' },
              { label: 'Integrations', slug: 'integrations' },
              { label: 'FAQs', slug: 'faq' },
          ],
          customCss: [
            "./src/styles/global.css",  
          ],
      }),
	],

	site: "https://kiln.sh",

  vite: {
    plugins: [tailwindcss()],
  },
});
