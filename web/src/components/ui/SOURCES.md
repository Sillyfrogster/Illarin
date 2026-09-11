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

Both sources use the MIT license. Copyright shadcn and Ashutoshx7 respectively.
Their licenses are kept here as LICENSE-shadcn and LICENSE-vengeance.
