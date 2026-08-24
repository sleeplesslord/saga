# Design

<!-- impeccable:design-schema 1 -->

## Direction

Saga's web interface is **Ravens — Thought & Memory**: a read-only board that surveys the project and returns the state of every saga without acting. The board carries Thought (what is happening now); the inspector carries Memory (what has happened). The reference is the raven's report — ink-dark basalt and runestone surfaces, gilded carved seals, runic signal marks, and restrained ember-moss-gold signal colors on a blue-tinted night ground. Familiar task names, search, filtering, and a ledger reading order always outrank metaphor; the Norse voice lives in the display type and the marks, not in coined jargon.

## Surface Scene

The interface is designed for a developer or agent operator checking a project during normal desk work. It uses an ink-dark night ground for long-session focus, with a single gilded hairline framing identity and selection. The dense task ledger is the workhorse; the inspector opens alongside it to carry the saga's memory — description, plan, dependencies, and history.

## Colors

The strategy is restrained: blue-tinted dark surfaces plus semantic signal colors drawn from ember, frost moss, and gilded gold.

- `--void: #07090d` — deepest night, the top rail and summary band
- `--basalt: #0b0e13` — main ground, dark runestone
- `--slate: #11151d` — lifted panels: inspector, task rows, inputs
- `--rune: #161b25` — raised surfaces: column head, hover, relations
- `--rune-2: #1b212c` — raised hover
- `--ink: #ece7d8` — primary text, bone
- `--bone: #d6d1c1` — secondary text
- `--muted: #9ba1ae` — muted, blue-tinted slate
- `--faint: #9097a8` — tertiary text, legible on every dark surface
- `--line: #232a37` — hairlines on dark
- `--edge: #2c3340` — stronger edges
- `--ember: #d24a3c` — blocked, blood ember
- `--moss: #5fb585` — ready, frost moss
- `--gold: #d9a441` — claimed, ember gold
- `--iron: #71788a` — complete / paused / wontdo, cold iron
- `--gild: #e0ad4a` — selection and active control, gilded
- `--sheen: #7c8fc4` — raven iridescence, used sparingly

## Typography

Two families with a deliberate role split. The carved-inscriptional display face carries Norse voice only at large sizes; the workhorse sans carries every dense, scannable surface.

- **Saga Sans** (DejaVu Sans, self-hosted) — body, UI, labels, and all dense scannable content including task titles. The ordinary reading and scanning face.
- **Saga Display** (Cinzel Bold, self-hosted, OFL) — large display only: page title, summary counts, inspector detail title, board section title, empty-state titles, and markdown headings. Never used below 14px; the board's task titles and relation titles are sans so the ledger stays scannable.
- **Monospace** (ui-monospace, Courier New) — data only: saga IDs, timestamps, clock, code blocks.

Labels (eyebrows, column heads, state tags, section headings) are set in sans, tracked uppercase — not monospace — so they read as carved runic inscriptions rather than terminal costume. Reading content on dark surfaces is compensated with slightly more line height (markdown body 1.7) and one step more weight where the face needs it.

## Layout

A fixed top rail (identity, project, live clock) sits above a two-column shell: a left sidebar (navigation, rune key, read-only notice) and a main column. The main column stacks a page head (title + search/filter/refresh tools), a summary band (ready / blocked / claimed counts), and a workspace. The workspace is a task ledger grid whose rows expose dependency flow through indentation and branch connectors; a detail inspector opens alongside the board. The inspector can expand to fullscreen for long reading. The layout collapses to a single column with a bottom-nav sidebar under 760px.

## Components

- **Brand seal** — a gilded ring bearing the Sowilo rune (ᛋ), set in Cinzel-tracked SAGA wordmark.
- **Lamps** — solid semantic-color marks with a dark inset ring (no glow); each paired with a word.
- **Task rows** — state mark, title, dependency summary, and relative time; selected rows take a gilded gradient and a 1px gilded hairline.
- **Rune key** — a runic mark (::before clip-path) labels the legend.
- **Inspector** — detail title, markdown description and plan, metadata grid, labeled relation buttons, and a timestamped history list.
- **Markdown** — full dark-theme rendering: code blocks, tables, blockquotes, task lists, inline code, links.

## Interaction and Motion

The inspector slides open (transform, 280ms cubic-bezier) and fades (opacity, 200ms). The refresh icon spins while loading. Selection updates are instant. All motion respects `prefers-reduced-motion`. Keyboard: `/` focuses search, `Escape` closes the inspector or exits fullscreen.

## Accessibility and Resilience

Every text element passes WCAG AA on its dark surface (verified: 0 low-contrast warnings across board and inspector). Focus rings are 2px gilded with offset. The board is keyboard-navigable; the inspector is focusable. `aria-live` regions announce list and toast updates. Fonts use `font-display: swap` with metric-compatible fallbacks (Georgia for display, Arial for sans) to avoid invisible text. The CSP restricts all sources to `'self'` (fonts are self-hosted to satisfy `font-src 'self'`). Reduced motion and user font scaling are preserved.
