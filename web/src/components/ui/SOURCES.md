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

- https://raw.githubusercontent.com/Ashutoshx7/VengeanceUI/main/public/r/notch-navbar.json
- https://raw.githubusercontent.com/Ashutoshx7/VengeanceUI/main/public/r/line-hover-link.json
- https://github.com/Ashutoshx7/VengeanceUI/blob/main/LICENSE

Both sources use the MIT license. Copyright shadcn and Ashutoshx7 respectively.
Their licenses are kept here as LICENSE-shadcn and LICENSE-vengeance.
