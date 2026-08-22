# Design

<!-- impeccable:design-schema 1 -->

## Direction

Saga's web interface is a **Signal Box**: a calm local dispatch surface inspired by railway interlocking boards and paper train orders. Tasks are routes, hard dependencies are signals, and state color answers whether work can move. The reference is operational rather than nostalgic; familiar task names, search, filtering, and a ledger reading order always outrank metaphor.

## Surface Scene

The interface is designed for a developer or agent operator checking a project during normal desk work in mixed ambient light. It uses a light paper work surface for long-session readability, with a dark painted control rail framing navigation and identity.

## Color

The strategy is restrained: warm neutral surfaces plus semantic signal colors.

- `--paper: #f2efe6` — main dispatch paper
- `--paper-2: #e4e1d7` — summary and secondary work surface
- `--panel: #fbfaf5` — ledger and inspector
- `--ink: #252822` — primary text
- `--muted: #6f726a` — supporting text
- `--control: #292c27` — sidebar
- `--control-2: #20231f` — top rail
- `--red: #b83e33` — blocked / danger
- `--green: #347b65` — ready / clear
- `--amber: #d1a02e` — claimed / occupied
- `--gray: #858a82` — completed, paused, wontdo
- `--gold: #c49c37` — selection and active control

Signal color is never the only state indicator; every lamp is paired with a word.

## Typography

Self-hosted DejaVu assets ship in the Go binary to avoid network dependencies.

- **Saga Display / DejaVu Serif Bold** — page titles, task titles, and numeric summaries; resembles durable printed dispatch documents.
- **Saga Sans / DejaVu Sans** — controls and body copy.
- **Courier New fallback stack** — IDs, timestamps, counts, and measured system labels only.

Task titles are concise and scannable. Supporting prose stays compact because this is an Operate surface, not a reading surface.

## Layout

- A fixed dark top rail names the project and local runtime.
- A narrow left control rail provides status-derived views.
- The main surface begins with search and status controls, followed by ready/blocked/claimed totals.
- A ledger is the primary visualization. Prerequisites are summarized inline rather than hidden in a separate graph tab.
- Selecting a saga opens a right inspector with full relationships, children, plan, and recent history.
- On narrow screens the control rail becomes a compact horizontal bar and the inspector becomes a slide-over sheet.

## Components

### Signal lamp

A 10px enamel-like circle plus a textual state label. Green means ready, red blocked, amber claimed, gray terminal or paused.

### Task row

A full-width ledger row with signal, hierarchical title and ID, dependency summary, and update time. Rows are buttons and support visible keyboard focus.

### Relationship row

A compact inspector control linking to a dependency, dependent, child, or related saga. It repeats the target's signal and context.

### Controls

Controls use restrained rectangular forms, one-pixel rules, and direct labels. Rounded pills, floating cards, decorative shadows, and detached metric tiles do not belong in this world.

## Interaction and Motion

- Search filters immediately; `/` focuses search.
- Status views and totals act as direct filters.
- Selecting a row updates the URL hash for a linkable inspection state.
- `Escape` closes the inspector.
- Data refreshes every 30 seconds and through an explicit Refresh button.
- The primary authored motion is the inspector deploying from the route ledger. Reduced-motion users receive the state change without transition.

## Accessibility and Resilience

- Native buttons, labels, search, select, navigation, and headings preserve keyboard and screen-reader behavior.
- Focus is visibly gold against both light and dark controls.
- Responsive layouts avoid horizontal page overflow.
- Empty, loading, storage-error, and no-result states use actionable copy.
- The server binds only to localhost, exposes GET routes only, sends restrictive security headers, and performs no storage mutations.
