# Component sources

Button, Popover, DropdownMenu and Sheet are adapted from shadcn/ui's New York
registry, retrieved 8 September 2026. The adaptations replace shadcn's default
tokens with the Studio tokens in `app/globals.css`, raise every target to 44px,
drop the parts of each registry file this site has no use for, and add a loading
state to Button. Enter and exit animations are gated on `motion-safe` rather than
overridden under reduced motion.

- https://ui.shadcn.com/r/styles/new-york/button.json
- https://ui.shadcn.com/r/styles/new-york/popover.json
- https://ui.shadcn.com/r/styles/new-york/dropdown-menu.json
- https://ui.shadcn.com/r/styles/new-york/sheet.json
- https://github.com/shadcn-ui/ui/blob/main/LICENSE.md

`ui/drawer.tsx` adapts shadcn/ui's drawer, retrieved 12 September 2026, which
wraps vaul. It keeps the bottom edge, the drag handle and drag to dismiss, drops
the other three edges and the header and footer parts, and uses the Studio
tokens. The global reduced-motion rule holds vaul's slide still.

- https://ui.shadcn.com/r/styles/new-york-v4/drawer.json

`layout/Notch.tsx` adapts Vengeance UI's notch navbar: its five-slice row, its
clip-path corner geometry and its taller centre section. It replaces the source's
two 5-per-cent outline strokes with one line at 3:1 that traces the whole
silhouette, paints the header's full height so page content cannot pass beside
the bay, and puts the mark and wordmark in the bay rather than in the rails.

`ui/line-link.tsx` adapts Vengeance UI's line hover link, keeping the `slide`
variant: the rule scales in from the left on hover and keyboard focus and
retracts to the right. It adds the current-page state and expresses the source's
stylesheet as utilities on one component.

`ui/fullscreen-preview.tsx` adapts Vengeance UI's fullscreen preview, keeping
its portal, its black backdrop, its Escape key and its scroll lock on the body.
It takes one picture rather than arbitrary children, moves focus to the close
button on opening and gives it back on closing, and raises that button to 44px.

`ui/copy-button.tsx` adapts Vengeance UI's copy button. It keeps the icon swap
and the timed confirmation, and adds a spoken confirmation, a failure state
when the clipboard refuses, and a 44px target.

`ui/morphing-disclosure.tsx` adapts Vengeance UI's morphing disclosure, keeping
its layout spring and the fade-and-rise of the panel, so a region whose contents
change while it is open settles into the new height rather than jumping. It
replaces the source's clickable div and text plus sign with a button that reports
`aria-expanded` and names its panel, gives the panel a lucide chevron, holds
the layout still under reduced motion, and adds a `trailing` slot so a caller can
read the name first and the explanation after it.

`ui/travelling-highlight.tsx` adapts Vengeance UI's highlight grid, keeping the
one plate that glides between cells and the sheen laid over it. It replaces the
source's cycled rainbow palette with the site's one action colour, tracks a
chosen cell as well as the pointed one so the plate returns to where you are
when the pointer leaves, follows keyboard focus, re-seats itself when the row
resizes, and skips the slide on its first placement.

- https://raw.githubusercontent.com/Ashutoshx7/VengeanceUI/main/public/r/notch-navbar.json
- https://raw.githubusercontent.com/Ashutoshx7/VengeanceUI/main/public/r/line-hover-link.json
- https://raw.githubusercontent.com/Ashutoshx7/VengeanceUI/main/public/r/fullscreen-preview.json
- https://raw.githubusercontent.com/Ashutoshx7/VengeanceUI/main/public/r/copy-button.json
- https://raw.githubusercontent.com/Ashutoshx7/VengeanceUI/main/public/r/morphing-disclosure.json
- https://raw.githubusercontent.com/Ashutoshx7/VengeanceUI/main/public/r/highlight-grid.json
- https://github.com/Ashutoshx7/VengeanceUI/blob/main/LICENSE

`ui/expanding-panel.tsx` adapts Cult UI's expandable screen, retrieved
12 September 2026: the trigger's background and the open surface share one
layout id, so the card grows into the panel and shrinks back. It puts a Radix
dialog underneath for focus, Escape and the scroll lock, sizes the open surface
to its content on a desktop rather than the whole window, keeps the trigger in
place while the panel is open, and holds still under reduced motion.

`ui/timeline.tsx` follows the shape of Dice UI's timeline, retrieved the same
day: a list of items, each with a dot, a connector and its content. It drops the
horizontal and alternating variants and the step status, and draws the
connector from one dot to the next.

- https://raw.githubusercontent.com/nolly-studio/cult-ui/main/apps/www/public/r/expandable-screen.json
- https://diceui.com/r/new-york/timeline.json

Every source uses the MIT license. Copyright shadcn, Ashutoshx7, Jordan-Gilliam
and Sadman Sakib respectively. Their licenses are kept here as LICENSE-shadcn,
LICENSE-vengeance, LICENSE-cult and LICENSE-dice.
