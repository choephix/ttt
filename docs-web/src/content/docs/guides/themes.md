---
title: Themes
description: Customizing the look and feel of TTT.
---

TTT supports fully customizable themes via JSON files. You can change every color in the editor, from syntax highlighting and diff backgrounds to the sidebar, tabs, status bar, terminal ANSI colors, borders, and semantic colors.

## Built-in Themes

10 themes are included:

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

To create a custom theme, copy one of the built-in theme files to your themes directory and edit it:

```sh
mkdir -p ~/.config/ttt/themes
cp sample-config/monokai.json ~/.config/ttt/themes/my-theme.json
```

Set it in `settings.json`:

```json
{
  "theme": "my-theme"
}
```

Restart TTT (or switch themes) to pick up changes.

To use your terminal's native colors instead of the theme's, set foreground/background to empty strings in your theme file.

## Theme Files

Theme files are stored as `<name>.json` in the `themes/` subdirectory of your config directory (`~/.config/ttt/themes/`). The filename (without `.json`) is the theme name used in `settings.json` and the theme picker.
