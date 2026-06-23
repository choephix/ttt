---
title: Extra Themes
description: Browse and install 78 extra themes for the TTT terminal text editor.
---

Beyond the 10 built-in themes, there are 78 extra themes available covering a wide range of aesthetics — from art-inspired palettes and city vibes to retro computing experiments. These themes are not bundled in the binary but are available in the repository for download.

## Installation

To install a theme, download the `.json` file and place it in `~/.config/ttt/themes/`. That's it — TTT will pick it up automatically in the theme picker (**Ctrl+K Ctrl+T**).

Each theme below shows a code (e.g. `19-synthwave`). Use it to download directly:

```sh
mkdir -p ~/.config/ttt/themes
curl -Lo ~/.config/ttt/themes/<theme>.json \
  https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/<theme>.json
```

Or grab all 78 at once:

```sh
git clone --depth 1 https://github.com/eugenioenko/ttt.git /tmp/ttt-themes
mkdir -p ~/.config/ttt/themes
cp /tmp/ttt-themes/config/themes/*.json ~/.config/ttt/themes/
rm -rf /tmp/ttt-themes
```

## Theme Gallery

<div class="theme-gallery">
  <div class="theme-card">
    <a href="/themes/01-vermeer.png" class="theme-preview"><img src="/themes/01-vermeer.png" alt="Vermeer" loading="lazy" /></a>
    <p>Vermeer</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/01-vermeer.json" class="theme-code-link"><code>01-vermeer</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/02-art-deco.png" class="theme-preview"><img src="/themes/02-art-deco.png" alt="Art Deco" loading="lazy" /></a>
    <p>Art Deco</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/02-art-deco.json" class="theme-code-link"><code>02-art-deco</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/03-impressionist.png" class="theme-preview"><img src="/themes/03-impressionist.png" alt="Impressionist" loading="lazy" /></a>
    <p>Impressionist</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/03-impressionist.json" class="theme-code-link"><code>03-impressionist</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/04-ukiyo-e.png" class="theme-preview"><img src="/themes/04-ukiyo-e.png" alt="Ukiyo-e" loading="lazy" /></a>
    <p>Ukiyo-e</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/04-ukiyo-e.json" class="theme-code-link"><code>04-ukiyo-e</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/05-bauhaus.png" class="theme-preview"><img src="/themes/05-bauhaus.png" alt="Bauhaus" loading="lazy" /></a>
    <p>Bauhaus</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/05-bauhaus.json" class="theme-code-link"><code>05-bauhaus</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/06-rothko.png" class="theme-preview"><img src="/themes/06-rothko.png" alt="Rothko" loading="lazy" /></a>
    <p>Rothko</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/06-rothko.json" class="theme-code-link"><code>06-rothko</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/07-deep-ocean.png" class="theme-preview"><img src="/themes/07-deep-ocean.png" alt="Deep Ocean" loading="lazy" /></a>
    <p>Deep Ocean</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/07-deep-ocean.json" class="theme-code-link"><code>07-deep-ocean</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/08-autumn-forest.png" class="theme-preview"><img src="/themes/08-autumn-forest.png" alt="Autumn Forest" loading="lazy" /></a>
    <p>Autumn Forest</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/08-autumn-forest.json" class="theme-code-link"><code>08-autumn-forest</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/09-arctic-tundra.png" class="theme-preview"><img src="/themes/09-arctic-tundra.png" alt="Arctic Tundra" loading="lazy" /></a>
    <p>Arctic Tundra</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/09-arctic-tundra.json" class="theme-code-link"><code>09-arctic-tundra</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/10-desert-sandstone.png" class="theme-preview"><img src="/themes/10-desert-sandstone.png" alt="Desert Sandstone" loading="lazy" /></a>
    <p>Desert Sandstone</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/10-desert-sandstone.json" class="theme-code-link"><code>10-desert-sandstone</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/11-tropical-rainforest.png" class="theme-preview"><img src="/themes/11-tropical-rainforest.png" alt="Tropical Rainforest" loading="lazy" /></a>
    <p>Tropical Rainforest</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/11-tropical-rainforest.json" class="theme-code-link"><code>11-tropical-rainforest</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/12-volcanic.png" class="theme-preview"><img src="/themes/12-volcanic.png" alt="Volcanic" loading="lazy" /></a>
    <p>Volcanic</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/12-volcanic.json" class="theme-code-link"><code>12-volcanic</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/13-tokyo-neon.png" class="theme-preview"><img src="/themes/13-tokyo-neon.png" alt="Tokyo Neon" loading="lazy" /></a>
    <p>Tokyo Neon</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/13-tokyo-neon.json" class="theme-code-link"><code>13-tokyo-neon</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/14-kyoto-zen.png" class="theme-preview"><img src="/themes/14-kyoto-zen.png" alt="Kyoto Zen" loading="lazy" /></a>
    <p>Kyoto Zen</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/14-kyoto-zen.json" class="theme-code-link"><code>14-kyoto-zen</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/15-havana-sunset.png" class="theme-preview"><img src="/themes/15-havana-sunset.png" alt="Havana Sunset" loading="lazy" /></a>
    <p>Havana Sunset</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/15-havana-sunset.json" class="theme-code-link"><code>15-havana-sunset</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/16-scandinavian.png" class="theme-preview"><img src="/themes/16-scandinavian.png" alt="Scandinavian" loading="lazy" /></a>
    <p>Scandinavian</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/16-scandinavian.json" class="theme-code-link"><code>16-scandinavian</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/17-marrakech.png" class="theme-preview"><img src="/themes/17-marrakech.png" alt="Marrakech" loading="lazy" /></a>
    <p>Marrakech</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/17-marrakech.json" class="theme-code-link"><code>17-marrakech</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/18-noir.png" class="theme-preview"><img src="/themes/18-noir.png" alt="Noir" loading="lazy" /></a>
    <p>Noir</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/18-noir.json" class="theme-code-link"><code>18-noir</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/19-synthwave.png" class="theme-preview"><img src="/themes/19-synthwave.png" alt="Synthwave" loading="lazy" /></a>
    <p>Synthwave</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/19-synthwave.json" class="theme-code-link"><code>19-synthwave</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/20-jazz-club.png" class="theme-preview"><img src="/themes/20-jazz-club.png" alt="Jazz Club" loading="lazy" /></a>
    <p>Jazz Club</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/20-jazz-club.json" class="theme-code-link"><code>20-jazz-club</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/21-punk-rock.png" class="theme-preview"><img src="/themes/21-punk-rock.png" alt="Punk Rock" loading="lazy" /></a>
    <p>Punk Rock</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/21-punk-rock.json" class="theme-code-link"><code>21-punk-rock</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/22-bossa-nova.png" class="theme-preview"><img src="/themes/22-bossa-nova.png" alt="Bossa Nova" loading="lazy" /></a>
    <p>Bossa Nova</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/22-bossa-nova.json" class="theme-code-link"><code>22-bossa-nova</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/23-lofi.png" class="theme-preview"><img src="/themes/23-lofi.png" alt="Lofi" loading="lazy" /></a>
    <p>Lofi</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/23-lofi.json" class="theme-code-link"><code>23-lofi</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/24-baroque.png" class="theme-preview"><img src="/themes/24-baroque.png" alt="Baroque" loading="lazy" /></a>
    <p>Baroque</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/24-baroque.json" class="theme-code-link"><code>24-baroque</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/25-sepia.png" class="theme-preview"><img src="/themes/25-sepia.png" alt="Sepia" loading="lazy" /></a>
    <p>Sepia</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/25-sepia.json" class="theme-code-link"><code>25-sepia</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/26-midnight-blue.png" class="theme-preview"><img src="/themes/26-midnight-blue.png" alt="Midnight Blue" loading="lazy" /></a>
    <p>Midnight Blue</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/26-midnight-blue.json" class="theme-code-link"><code>26-midnight-blue</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/27-emerald.png" class="theme-preview"><img src="/themes/27-emerald.png" alt="Emerald" loading="lazy" /></a>
    <p>Emerald</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/27-emerald.json" class="theme-code-link"><code>27-emerald</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/28-rose.png" class="theme-preview"><img src="/themes/28-rose.png" alt="Rose" loading="lazy" /></a>
    <p>Rose</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/28-rose.json" class="theme-code-link"><code>28-rose</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/29-slate.png" class="theme-preview"><img src="/themes/29-slate.png" alt="Slate" loading="lazy" /></a>
    <p>Slate</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/29-slate.json" class="theme-code-link"><code>29-slate</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/30-copper.png" class="theme-preview"><img src="/themes/30-copper.png" alt="Copper" loading="lazy" /></a>
    <p>Copper</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/30-copper.json" class="theme-code-link"><code>30-copper</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/31-nebula.png" class="theme-preview"><img src="/themes/31-nebula.png" alt="Nebula" loading="lazy" /></a>
    <p>Nebula</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/31-nebula.json" class="theme-code-link"><code>31-nebula</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/32-mars-colony.png" class="theme-preview"><img src="/themes/32-mars-colony.png" alt="Mars Colony" loading="lazy" /></a>
    <p>Mars Colony</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/32-mars-colony.json" class="theme-code-link"><code>32-mars-colony</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/33-cherry-blossom.png" class="theme-preview"><img src="/themes/33-cherry-blossom.png" alt="Cherry Blossom" loading="lazy" /></a>
    <p>Cherry Blossom</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/33-cherry-blossom.json" class="theme-code-link"><code>33-cherry-blossom</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/34-obsidian.png" class="theme-preview"><img src="/themes/34-obsidian.png" alt="Obsidian" loading="lazy" /></a>
    <p>Obsidian</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/34-obsidian.json" class="theme-code-link"><code>34-obsidian</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/35-jade.png" class="theme-preview"><img src="/themes/35-jade.png" alt="Jade" loading="lazy" /></a>
    <p>Jade</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/35-jade.json" class="theme-code-link"><code>35-jade</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/36-amethyst.png" class="theme-preview"><img src="/themes/36-amethyst.png" alt="Amethyst" loading="lazy" /></a>
    <p>Amethyst</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/36-amethyst.json" class="theme-code-link"><code>36-amethyst</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/37-coral-reef.png" class="theme-preview"><img src="/themes/37-coral-reef.png" alt="Coral Reef" loading="lazy" /></a>
    <p>Coral Reef</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/37-coral-reef.json" class="theme-code-link"><code>37-coral-reef</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/38-thunderstorm.png" class="theme-preview"><img src="/themes/38-thunderstorm.png" alt="Thunderstorm" loading="lazy" /></a>
    <p>Thunderstorm</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/38-thunderstorm.json" class="theme-code-link"><code>38-thunderstorm</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/39-espresso.png" class="theme-preview"><img src="/themes/39-espresso.png" alt="Espresso" loading="lazy" /></a>
    <p>Espresso</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/39-espresso.json" class="theme-code-link"><code>39-espresso</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/40-matcha.png" class="theme-preview"><img src="/themes/40-matcha.png" alt="Matcha" loading="lazy" /></a>
    <p>Matcha</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/40-matcha.json" class="theme-code-link"><code>40-matcha</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/41-polaroid.png" class="theme-preview"><img src="/themes/41-polaroid.png" alt="Polaroid" loading="lazy" /></a>
    <p>Polaroid</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/41-polaroid.json" class="theme-code-link"><code>41-polaroid</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/42-darkroom.png" class="theme-preview"><img src="/themes/42-darkroom.png" alt="Darkroom" loading="lazy" /></a>
    <p>Darkroom</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/42-darkroom.json" class="theme-code-link"><code>42-darkroom</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/43-northern-lights.png" class="theme-preview"><img src="/themes/43-northern-lights.png" alt="Northern Lights" loading="lazy" /></a>
    <p>Northern Lights</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/43-northern-lights.json" class="theme-code-link"><code>43-northern-lights</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/44-solar-flare.png" class="theme-preview"><img src="/themes/44-solar-flare.png" alt="Solar Flare" loading="lazy" /></a>
    <p>Solar Flare</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/44-solar-flare.json" class="theme-code-link"><code>44-solar-flare</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/45-vaporwave.png" class="theme-preview"><img src="/themes/45-vaporwave.png" alt="Vaporwave" loading="lazy" /></a>
    <p>Vaporwave</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/45-vaporwave.json" class="theme-code-link"><code>45-vaporwave</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/46-cyberpunk.png" class="theme-preview"><img src="/themes/46-cyberpunk.png" alt="Cyberpunk" loading="lazy" /></a>
    <p>Cyberpunk</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/46-cyberpunk.json" class="theme-code-link"><code>46-cyberpunk</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/47-art-nouveau.png" class="theme-preview"><img src="/themes/47-art-nouveau.png" alt="Art Nouveau" loading="lazy" /></a>
    <p>Art Nouveau</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/47-art-nouveau.json" class="theme-code-link"><code>47-art-nouveau</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/48-gotham.png" class="theme-preview"><img src="/themes/48-gotham.png" alt="Gotham" loading="lazy" /></a>
    <p>Gotham</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/48-gotham.json" class="theme-code-link"><code>48-gotham</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/49-mumbai-monsoon.png" class="theme-preview"><img src="/themes/49-mumbai-monsoon.png" alt="Mumbai Monsoon" loading="lazy" /></a>
    <p>Mumbai Monsoon</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/49-mumbai-monsoon.json" class="theme-code-link"><code>49-mumbai-monsoon</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/50-patagonia.png" class="theme-preview"><img src="/themes/50-patagonia.png" alt="Patagonia" loading="lazy" /></a>
    <p>Patagonia</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/50-patagonia.json" class="theme-code-link"><code>50-patagonia</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/51-glacier.png" class="theme-preview"><img src="/themes/51-glacier.png" alt="Glacier" loading="lazy" /></a>
    <p>Glacier</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/51-glacier.json" class="theme-code-link"><code>51-glacier</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/52-firefly.png" class="theme-preview"><img src="/themes/52-firefly.png" alt="Firefly" loading="lazy" /></a>
    <p>Firefly</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/52-firefly.json" class="theme-code-link"><code>52-firefly</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/53-steampunk.png" class="theme-preview"><img src="/themes/53-steampunk.png" alt="Steampunk" loading="lazy" /></a>
    <p>Steampunk</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/53-steampunk.json" class="theme-code-link"><code>53-steampunk</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/54-twilight.png" class="theme-preview"><img src="/themes/54-twilight.png" alt="Twilight" loading="lazy" /></a>
    <p>Twilight</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/54-twilight.json" class="theme-code-link"><code>54-twilight</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/55-campfire.png" class="theme-preview"><img src="/themes/55-campfire.png" alt="Campfire" loading="lazy" /></a>
    <p>Campfire</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/55-campfire.json" class="theme-code-link"><code>55-campfire</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/56-lavender.png" class="theme-preview"><img src="/themes/56-lavender.png" alt="Lavender" loading="lazy" /></a>
    <p>Lavender</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/56-lavender.json" class="theme-code-link"><code>56-lavender</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/57-submarine.png" class="theme-preview"><img src="/themes/57-submarine.png" alt="Submarine" loading="lazy" /></a>
    <p>Submarine</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/57-submarine.json" class="theme-code-link"><code>57-submarine</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/58-terracotta.png" class="theme-preview"><img src="/themes/58-terracotta.png" alt="Terracotta" loading="lazy" /></a>
    <p>Terracotta</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/58-terracotta.json" class="theme-code-link"><code>58-terracotta</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/59-experiment-double-line.png" class="theme-preview"><img src="/themes/59-experiment-double-line.png" alt="Experiment: Double Line" loading="lazy" /></a>
    <p>Experiment: Double Line</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/59-experiment-double-line.json" class="theme-code-link"><code>59-experiment-double-line</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/60-experiment-rounded.png" class="theme-preview"><img src="/themes/60-experiment-rounded.png" alt="Experiment: Rounded" loading="lazy" /></a>
    <p>Experiment: Rounded</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/60-experiment-rounded.json" class="theme-code-link"><code>60-experiment-rounded</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/61-experiment-brutalist.png" class="theme-preview"><img src="/themes/61-experiment-brutalist.png" alt="Experiment: Brutalist" loading="lazy" /></a>
    <p>Experiment: Brutalist</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/61-experiment-brutalist.json" class="theme-code-link"><code>61-experiment-brutalist</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/62-experiment-bbs.png" class="theme-preview"><img src="/themes/62-experiment-bbs.png" alt="Experiment: BBS" loading="lazy" /></a>
    <p>Experiment: BBS</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/62-experiment-bbs.json" class="theme-code-link"><code>62-experiment-bbs</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/63-experiment-dotted.png" class="theme-preview"><img src="/themes/63-experiment-dotted.png" alt="Experiment: Dotted" loading="lazy" /></a>
    <p>Experiment: Dotted</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/63-experiment-dotted.json" class="theme-code-link"><code>63-experiment-dotted</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/64-experiment-zen.png" class="theme-preview"><img src="/themes/64-experiment-zen.png" alt="Experiment: Zen" loading="lazy" /></a>
    <p>Experiment: Zen</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/64-experiment-zen.json" class="theme-code-link"><code>64-experiment-zen</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/65-experiment-monochrome-blue.png" class="theme-preview"><img src="/themes/65-experiment-monochrome-blue.png" alt="Experiment: Monochrome Blue" loading="lazy" /></a>
    <p>Experiment: Monochrome Blue</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/65-experiment-monochrome-blue.json" class="theme-code-link"><code>65-experiment-monochrome-blue</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/66-experiment-phosphor-green.png" class="theme-preview"><img src="/themes/66-experiment-phosphor-green.png" alt="Experiment: Phosphor Green" loading="lazy" /></a>
    <p>Experiment: Phosphor Green</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/66-experiment-phosphor-green.json" class="theme-code-link"><code>66-experiment-phosphor-green</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/67-experiment-phosphor-amber.png" class="theme-preview"><img src="/themes/67-experiment-phosphor-amber.png" alt="Experiment: Phosphor Amber" loading="lazy" /></a>
    <p>Experiment: Phosphor Amber</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/67-experiment-phosphor-amber.json" class="theme-code-link"><code>67-experiment-phosphor-amber</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/68-experiment-glitch.png" class="theme-preview"><img src="/themes/68-experiment-glitch.png" alt="Experiment: Glitch" loading="lazy" /></a>
    <p>Experiment: Glitch</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/68-experiment-glitch.json" class="theme-code-link"><code>68-experiment-glitch</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/69-experiment-blueprint.png" class="theme-preview"><img src="/themes/69-experiment-blueprint.png" alt="Experiment: Blueprint" loading="lazy" /></a>
    <p>Experiment: Blueprint</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/69-experiment-blueprint.json" class="theme-code-link"><code>69-experiment-blueprint</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/70-experiment-thermal.png" class="theme-preview"><img src="/themes/70-experiment-thermal.png" alt="Experiment: Thermal" loading="lazy" /></a>
    <p>Experiment: Thermal</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/70-experiment-thermal.json" class="theme-code-link"><code>70-experiment-thermal</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/71-experiment-gameboy.png" class="theme-preview"><img src="/themes/71-experiment-gameboy.png" alt="Experiment: Gameboy" loading="lazy" /></a>
    <p>Experiment: Gameboy</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/71-experiment-gameboy.json" class="theme-code-link"><code>71-experiment-gameboy</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/72-experiment-c64.png" class="theme-preview"><img src="/themes/72-experiment-c64.png" alt="Experiment: C64" loading="lazy" /></a>
    <p>Experiment: C64</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/72-experiment-c64.json" class="theme-code-link"><code>72-experiment-c64</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/73-experiment-redacted.png" class="theme-preview"><img src="/themes/73-experiment-redacted.png" alt="Experiment: Redacted" loading="lazy" /></a>
    <p>Experiment: Redacted</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/73-experiment-redacted.json" class="theme-code-link"><code>73-experiment-redacted</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/74-experiment-oscilloscope.png" class="theme-preview"><img src="/themes/74-experiment-oscilloscope.png" alt="Experiment: Oscilloscope" loading="lazy" /></a>
    <p>Experiment: Oscilloscope</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/74-experiment-oscilloscope.json" class="theme-code-link"><code>74-experiment-oscilloscope</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/75-experiment-infrared.png" class="theme-preview"><img src="/themes/75-experiment-infrared.png" alt="Experiment: Infrared" loading="lazy" /></a>
    <p>Experiment: Infrared</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/75-experiment-infrared.json" class="theme-code-link"><code>75-experiment-infrared</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/76-experiment-woodgrain.png" class="theme-preview"><img src="/themes/76-experiment-woodgrain.png" alt="Experiment: Woodgrain" loading="lazy" /></a>
    <p>Experiment: Woodgrain</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/76-experiment-woodgrain.json" class="theme-code-link"><code>76-experiment-woodgrain</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/77-experiment-neon-sign.png" class="theme-preview"><img src="/themes/77-experiment-neon-sign.png" alt="Experiment: Neon Sign" loading="lazy" /></a>
    <p>Experiment: Neon Sign</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/77-experiment-neon-sign.json" class="theme-code-link"><code>77-experiment-neon-sign</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
  <div class="theme-card">
    <a href="/themes/78-experiment-acid.png" class="theme-preview"><img src="/themes/78-experiment-acid.png" alt="Experiment: Acid" loading="lazy" /></a>
    <p>Experiment: Acid</p>
    <a href="https://raw.githubusercontent.com/eugenioenko/ttt/main/config/themes/78-experiment-acid.json" class="theme-code-link"><code>78-experiment-acid</code></a><button class="theme-copy" title="Copy theme name">&#x2398;</button>
  </div>
</div>

<script>
  (function() {
    var lb = document.createElement('div');
    lb.className = 'theme-lightbox';
    lb.id = 'theme-lightbox';
    lb.innerHTML = '<img src="" alt="" />';
    document.body.appendChild(lb);

    document.addEventListener('click', function(e) {
      var link = e.target.closest('.theme-preview');
      if (link) {
        e.preventDefault();
        lb.querySelector('img').src = link.href;
        lb.querySelector('img').alt = link.querySelector('img').alt;
        lb.classList.add('active');
      }
    });
    lb.addEventListener('click', function() {
      lb.classList.remove('active');
    });
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') lb.classList.remove('active');
    });

    document.addEventListener('click', function(e) {
      var btn = e.target.closest('.theme-copy');
      if (btn) {
        var code = btn.previousElementSibling.querySelector('code').textContent;
        navigator.clipboard.writeText(code);
        btn.textContent = '✓';
        setTimeout(function() { btn.innerHTML = '⎘'; }, 1500);
      }
    });
  })();
</script>
