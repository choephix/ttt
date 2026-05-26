---
title: Themes
description: Customizing the look and feel of TTT.
---

TTT supports fully customizable themes via JSON files. You can change every color in the editor, from syntax highlighting and diff backgrounds to the sidebar, tabs, status bar, terminal ANSI colors, borders, and semantic colors.

## Built-in Themes

10 themes ship in the `sample-config/` directory:

- Aurora
- Bubblegum
- Default Dark
- Default Light
- Hotline
- Monokai
- One Dark
- Solarized Dark
- Solarized Light
- Virtru Dark

## Switching Themes

Press **Ctrl+K Ctrl+T** (or use **View > Switch Theme** from the menu bar) to open the theme picker with a live preview.

## Customizing

To create a custom theme, copy one of the built-in theme files to your config directory and edit it:

```sh
cp sample-config/theme.monokai.json ~/.config/ttt/theme.json
```

Restart TTT (or switch themes) to pick up changes.

To use your terminal's native colors instead of the theme's, set foreground/background to empty strings in your theme file.

## Theme Setting

Set the active theme in `settings.json`:

```json
{
  "theme": "monokai"
}
```

Theme files are named `theme.<name>.json` in the config directory.
