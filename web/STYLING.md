# Interface styling

`src/app/globals.css` is the only host stylesheet. It defines the theme,
Tailwind tokens, shared animation keyframes and document defaults. Fonts load
through `src/lib/fonts.ts`. Page and component presentation belongs in Tailwind
and the shared components, with `cn` resolving class overrides.

Use `Shell` for the site content column. When an existing component or semantic
element owns the wrapper, use its exported `shellClasses`. The shell, gutter,
header and rail variables keep content aligned across breakpoints.

## Document defaults

The central reset is deliberate; Tailwind Preflight is not imported. It keeps
the existing box sizing, zero margins and padding, native heading and list
semantics, button inheritance and block images. Form controls and fieldsets
have no browser bevel or groove. Shared controls supply their own treatment.

Document rules set the background, prose font, link fallback, caret, selection,
visible keyboard focus and scrollbars. The main landmark fills the viewport
below the header. Reduced motion limits CSS transitions and keyframes, while
the root MotionConfig handles Framer Motion. Tailwind supplies `sr-only`.

Utilities are unlayered to retain their precedence over the document defaults.
Use flex or grid gaps for grouped content: the universal margin reset overrides
Tailwind's zero-specificity `space-y` rules.

## Library integration and dynamic values

- Radix portals inherit the document theme. Their shared components use Radix
  sizing and transform-origin variables through Tailwind utilities.
- `publication/writing/writing-surface.ts` applies shared utility rules to
  ProseMirror's generated document, selections and task-list controls.
- Framer Motion supplies changing transforms, opacity and scroll elevation.
  The travelling highlight measures its destination; resizable text fields and
  the asset workspace measure content and enforce the block arrangement rules.
- Artwork masks, SVG paths, media aspect ratios and generated art variables
  remain with the artwork or the component that consumes it.
- `lib/publication-card.tsx` uses inline styles for Next ImageResponse's image
  renderer, which does not load the browser's stylesheet.

Creator-authored stylesheets and colour values are asset content. They are not
host styles and must survive imports, editing and exports unchanged.

The retired design prototypes have no routes or stylesheets. Their artwork and
licences remain in `src/app/prototype/direction`; `make direction-fixtures`
can reproduce the synthetic artwork there. Production motion uses Framer Motion;
the landing film needs no browser 3D runtime.
