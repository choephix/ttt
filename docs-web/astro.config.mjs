import { defineConfig, passthroughImageService } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://eugenioenko.github.io',
  base: '/ttt',
  image: { service: passthroughImageService() },
  integrations: [
    starlight({
      customCss: ['./src/styles/custom.css'],
      title: 'TTT',
      favicon: '/favicon.svg',
      logo: {
        src: './src/assets/logo.svg',
      },
      description: 'A terminal text editor. A real alternative to VS Code, Zed, and Sublime that runs in your terminal. Single Go binary, zero config.',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/eugenioenko/ttt' },
      ],
      sidebar: [
        {
          label: 'Getting Started',
          items: [{ autogenerate: { directory: 'getting-started' } }],
        },
        {
          label: 'Guides',
          items: [
            { label: 'Editor Basics', link: '/guides/editor/' },
            { label: 'File Explorer & Workspaces', link: '/guides/workspaces/' },
            { label: 'Search', link: '/guides/search/' },
            { label: 'Git Integration', link: '/guides/git/' },
            { label: 'Integrated Terminal', link: '/guides/terminal/' },
            { label: 'LSP', link: '/guides/lsp/' },
            { label: 'Themes', link: '/guides/themes/' },
            { label: 'Keybindings', link: '/guides/keybindings/' },
          ],
        },
        {
          label: 'Reference',
          items: [{ autogenerate: { directory: 'reference' } }],
        },
      ],
    }),
  ],
});
