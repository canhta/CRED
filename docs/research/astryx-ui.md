# Astryx as the CRED console UI layer

Research on Meta's Astryx design system, commissioned after the operator
mandated it in place of the previously-planned shadcn/ui + Tailwind for CRED's
web console (Vite + React + TypeScript + Vitest, TanStack Query + TanStack
Router). The console must build to plain static assets embeddable in the Go
binary via `go:embed`, with a CGO-free backend.

Evidence standard per `.claude/rules/docs.md`: every external claim is labeled
VERIFIED (fetched URL / read source file / ran command) or UNVERIFIED, with a
source. Primary sources are the live site and the `facebook/astryx` repo read
through `gh`/raw.githubusercontent; marketing copy is labeled as a *claim*, not
proof of behavior.

## 0. What Astryx is (verification of the basics)

- **VERIFIED** — The repo exists: `facebook/astryx`, MIT, description "An open
  source design system that's fully customizable and agent ready", homepage
  `astryx.atmeta.com`. As of 2026-07-21 via `gh api repos/facebook/astryx`:
  **9,686 stars, 716 forks, 272 open issues** (open-issues count includes PRs),
  created **2026-01-09**, last push **2026-07-21**.
- **VERIFIED** — React + StyleX, "150+ accessible components", Beta.
  Source: repo `README.md` (raw) and `https://astryx.atmeta.com/` (fetched).
  Component *counts differ by source* — the site landing says "over 160", the
  README and docs say "150+". Treat "~150" as the honest figure; the exact
  number is marketing-variable, not load-bearing.
- **VERIFIED** — Packages are scoped `@astryxdesign/*`. Workspace `packages/`
  dir lists: `build, charts, cli, core, lab, themes, vega`
  (`gh api .../contents/packages`). Core is `@astryxdesign/core` **v0.1.7**
  (`packages/core/package.json`).
- Secondary/marketing (label as claims): "grown inside Meta over eight years",
  "powers 13,000+ apps" — site landing + press
  ([MarkTechPost 2026-06-27](https://www.marktechpost.com/2026/06/27/metas-astryx-brings-a-cli-and-mcp-server-to-an-open-source-react-design-system-agents-can-read/)).
  Unverifiable from outside; not relied on here.

## (a) Install + concrete Vite/StyleX config

There are **two distinct setups**. This is the single most important finding:
Astryx does **not** require you to compile StyleX unless you opt in.

### Path A — pre-built CSS (recommended for CRED). No StyleX build at all.

**VERIFIED** — `packages/core/README.md` Quick Start and `theme-neutral/README.md`:
Astryx ships pre-built CSS + JS; consumers import three stylesheets and a theme
provider. README quote: "You import pre-built CSS and use typed React
components — **no build plugin, no styling library to adopt**." The theme
`/built` import path is documented as "Pre-built dist … or no build step".

```bash
npm install @astryxdesign/core @astryxdesign/theme-neutral
# React 19 + react-dom 19 are peer deps (see §d gotchas)
```

Global CSS (order matters — it maps to the `@layer` cascade):

```css
@import '@astryxdesign/core/reset.css';        /* @layer reset */
@import '@astryxdesign/core/astryx.css';        /* @layer astryx-base */
@import '@astryxdesign/theme-neutral/theme.css'; /* @layer astryx-theme */
```

Provider (names verified from source; `XDSTheme` and `Theme` both appear):

```tsx
import {XDSTheme} from '@astryxdesign/core/theme';
import {neutralTheme} from '@astryxdesign/theme-neutral/built';

export function App({children}) {
  return <XDSTheme theme={neutralTheme}>{children}</XDSTheme>;
}
```

With Path A there is **no `vite.config.ts` change, no Babel, no PostCSS, no
StyleX plugin.** It is ordinary React + CSS. `vite build` emits static
JS/CSS/asset files — exactly what `go:embed` needs. **UNVERIFIED** (not built
here, but nothing in the setup opposes it): a stock Vite SPA build produces the
same `dist/` regardless of Astryx; the only inputs are a React component library
and static CSS files.

### Path B — StyleX source compilation (opt-in; skip unless you need it)

**VERIFIED** — `apps/example-vite/vite.config.ts` (raw) is the official Vite
example and uses Path B:

```ts
import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import {astryxStylex} from '@astryxdesign/build/vite';

export default defineConfig({
  plugins: [...astryxStylex(), react()],  // StyleX plugin BEFORE react()
});
```

**VERIFIED** — `apps/example-vite/package.json` deps for this path:
`@stylexjs/stylex ^0.19`, and devDeps `@stylexjs/unplugin ^0.19`,
`@astryxdesign/build`, `@vitejs/plugin-react ^5.2`, `vite ^8.1`, `typescript ^6`,
`react/react-dom ^19.2`. `@astryxdesign/build/vite` "wraps unplugin + splits
layers" (`packages/build/README.md`).

**VERIFIED** — What Path B buys you (`packages/build/README.md`): it compiles
library code with an `astryx` class prefix and *your* product's StyleX with the
default `x` prefix, into separate `@layer`s
(`reset < astryx-base < astryx-theme < product`). This matters only if **you
also author styles in StyleX** and need your overrides layered against Astryx's,
or you want tree-shaken CSS (README claims "~33% of the full stylesheet" in
reference apps). For a console that overrides via `className`/plain CSS, Path A's
full pre-built stylesheet is simpler and the size delta is not worth the Babel +
unplugin toolchain.

Underlying official StyleX Vite plugin, for reference: **`@stylexjs/unplugin`**,
used as `stylex.vite({useCSSLayers: true})` before `@vitejs/plugin-react`, with a
CSS file imported from app root so Vite emits an asset; during `vite build` the
plugin appends aggregated CSS to that asset. Source:
[stylexjs.com Vite+React install](https://stylexjs.com/docs/learn/installation/vite/vite-react)
(VERIFIED, fetched). (`vite-plugin-stylex` by HorusGoul and `unplugin-stylex` by
eryue0220 are older community plugins — not what Astryx uses.)

**SSR / Next.js assumptions that do NOT bind a pure SPA:** Astryx's docsite and
several examples are Next.js, and the README's "simplest" walkthrough is Next.js
(App Router, `'use client'`, `next/link` via `LinkProvider`). None of that is
required — there is a first-class `example-vite` and `example-vite-tailwind`.
The Next-only pieces (`LinkProvider component={Link}`, `transpilePackages`,
`next.config` webpack `conditionNames`) are framework glue we replace with the
plain Path A provider above. **VERIFIED** from `apps/` listing and both READMEs.

**Recommendation for CRED: Path A.** Adopt `@astryxdesign/core` +
`@astryxdesign/theme-neutral` (or a custom theme, §c), three CSS imports, one
provider. No change to `vite.config.ts`, no StyleX plugin, no Babel/PostCSS —
which keeps the static-asset build and the Go embedding path untouched.

## (b) Component inventory vs console needs

Core exports were read from `packages/core/package.json` `exports` and the
Table source dir; the component pages/Storybook corroborate. Console needs from
the brief: claims data tables, forms, usage/limit views, analytics+charts, team
management, SSO/login, project management, file upload.

| Console need | Astryx ships | Status | Source |
|---|---|---|---|
| Button / actions | Button, ButtonGroup, IconButton, ToggleButton(Group) | Covered | components page (VERIFIED) |
| Card | Card, ClickableCard, SelectableCard | Covered | components page |
| Text input | TextInput, NumberInput, FileInput | Covered | components page |
| Select / combobox | Selector, MultiSelector, **Typeahead** (= combobox), BaseTypeahead | Covered | components page |
| Checkbox / radio | CheckboxInput, CheckboxList(Item), RadioList(Item) | Covered | components page |
| Date / time picker | Calendar, DateInput, DateRangeInput, DateTimeInput, TimeInput | Covered | components page |
| Dialog / modal | Dialog, AlertDialog, DialogHeader | Covered | components page |
| Toast | Toast + `useToast` | Covered | components page |
| Tabs | Tab, TabList, TabMenu | Covered | components page |
| Menu | DropdownMenu, ContextMenu, MoreMenu, TopNavMegaMenu | Covered | components page |
| App shell / nav | AppShell, SideNav, TopNav, MobileNav, Breadcrumbs | Covered | components page |
| Pagination | Pagination | Covered | components page |
| **Data table** | **Table** with plugin hooks (see below) | Covered — strong | `Table/` source (VERIFIED) |
| File upload | FileInput | Covered (basic) | components page |
| **Charts / data-viz** | `@astryxdesign/charts` (+ `@astryxdesign/vega`) | **Gap — pre-release** | `packages/charts/README.md` |

**Table — the standout, and it changes our stack.** **VERIFIED** from
`packages/core/src/Table/` (`Table.doc.mjs` + hook files): the Table is not just
presentational — it ships a **composable plugin/hook layer** covering the exact
concerns we'd otherwise pull TanStack Table for:
`useTableSortable`, `useTablePagination`, `useTableFiltering` /
`useTableFilterState`, `useTableSelection` / `useTableSelectionState`,
`useTableColumnResize`, `useTableColumnSettings`, `useTableStickyColumns`,
`useTableGroupedRows`, `useTableRowExpansion`. Doc keywords include
`"virtualized"`. Slots: Column Header (sorting + bulk select), Top Bar
(toolbar/filters), Bottom Bar (pagination). **For CRED this means the Astryx
Table likely covers the claims grid without adding TanStack Table.** Caveat
(UNVERIFIED): I did not build against these hooks; confirm server-driven
(manual) pagination/sorting wiring to TanStack Query before committing — if the
plugins are client-state-only, pair with TanStack Table for server-side data.

**Charts — the real gap.** **VERIFIED** `packages/charts/README.md`:
`@astryxdesign/charts` is a d3-based config-model lib (`<Chart>` with `bar`,
`line`, `area`, `dot`, `candlestick`, `errorBar`, `referenceLine` marks, shared
scales). But: "It ships to npm **only under the `@canary` dist-tag — there is
never a stable (`latest`) release yet**", `package.json` is `private:true` +
`canaryOnly`, and status is "API and visuals are still being refined." A second
package `@astryxdesign/vega` exists (Vega-based). **Verdict:** do not build
analytics on Astryx charts now. **Companion lib: Recharts** (simplest, React-19
compatible, covers line/bar/area/pie for usage/limit views) — or **visx** if we
need low-level d3 control. Revisit `@astryxdesign/charts` when it cuts a stable
release. This is the one console surface Astryx does not currently serve.

Minor: `FileInput` covers file selection; drag-drop/upload-progress UX may need
custom composition (UNVERIFIED — depth of FileInput not inspected). SSO/login is
just forms + layout, fully covered.

## (c) Theming, dark mode, tokens

- **VERIFIED** (`theme.doc.dense.mjs`) — Themes are token cascades of CSS custom
  properties in `@layer astryx-theme`. Published themes: `neutral` (start here),
  `butter, chocolate, gothic (dark-only), matcha, stone, y2k`.
- **VERIFIED** — Custom brand without forking components: **`defineTheme`**.
  Doc: "scale configs (color, typography, radius, motion) + explicit token
  overrides + component overrides. color derives full palette from accent hex
  via HCT." So a CRED theme = supply a brand accent hex (+ optional overrides);
  Astryx generates the palette. Component-level restyle is via a `components`
  field using "semantic component keys + style keys (base, variant:value,
  stateName)", or external CSS via documented `data-*` selectors — **not** by
  editing component source. `swizzle` (§e) is the escape hatch for deeper
  changes.
- **VERIFIED** — Production: `npx astryx theme build` compiles a `defineTheme`
  file to static `.css` + `.js` + `.d.ts`. Fits the static-asset build.
- **VERIFIED** — Dark mode (`theme.doc.dense.mjs`, `MediaTheme.doc.mjs`):
  page-wide dark mode is the `Theme`/`XDSTheme` prop **`mode="dark" |
  "light" | "system"`** (`system` follows OS). Token values carry `[light,
  dark]` tuples compiled to CSS `light-dark()`. Runtime toggle = drive the
  `mode` prop from app state. (`MediaTheme` is a separate, local surface-
  inversion primitive for overlays/toasts — not the app-level switch.)

Assessment: theming is more capable than shadcn's copy-and-edit model — brand
palette from one hex, live light/dark, component overrides through tokens, no
fork required.

## (d) TypeScript + Vitest testing

- **VERIFIED** — Fully typed: repo is TypeScript-first (`gh` language stats
  ~75% TS), core ships `.d.ts`, peer deps require `react >=19` typings.
- **VERIFIED** — Astryx tests itself with **Vitest**: repo root has
  `vitest.config.ts` and `vitest.global-setup.node.mjs`; the Table dir contains
  `Table.test.tsx`, `Table.perf.test.tsx`, `tableContextMenu.test.tsx`. So
  Vitest + RTL against Astryx components is the maintainers' own path.
- **The StyleX testing gotcha — and why Path A dodges it.** StyleX compiles
  classNames at build time. Its Babel plugin has a **`test: true`** option that
  "only outputs debug classNames … does not generate functional classNames …
  useful for snapshot testing" (**VERIFIED** via Context7 `/facebook/stylex`
  babel-plugin docs). That option matters **only if you compile StyleX source in
  the test pipeline (Path B).** On **Path A** we import Astryx's *pre-compiled*
  JS + static CSS: there is no StyleX transform in our Vitest run, so there is
  nothing to no-op. Components render in jsdom with stable (already-compiled)
  className strings; RTL queries by role/text/label work normally. jsdom does no
  layout/CSS cascade regardless, so visual styling is never asserted in unit
  tests either way.
- **UNVERIFIED (low risk):** I did not execute a Vitest run against
  `@astryxdesign/core` in a fresh Vite app. What would check it: scaffold Path A,
  render a `Button`/`Dialog`, assert by role. Watch for two real (not StyleX-
  specific) snags: (1) the package ships ESM/modern JS — ensure Vitest
  `deps`/transform handles `@astryxdesign/*` (usually automatic with Vite), and
  (2) React 19 requires `@testing-library/react` ≥16.

## (e) CLI + MCP

- **VERIFIED** (`packages/core/README.md`) — `@astryxdesign/cli` commands:
  `init`, `component <Name>` (props + examples), `search`, `docs [topic]`,
  `template <name>` (inject page/block templates), `hook`, **`swizzle <Name>`**,
  `upgrade --apply` (codemods), `theme build`, `discover`, `doctor`,
  `gap-report`. Every command supports `--dense` for token-efficient AI output.
- **Distribution model — package, not vendored.** **VERIFIED**: default is
  importing from `@astryxdesign/core`. `swizzle` is the opt-in shadcn-style eject
  ("Copy component source into your project for deep customization"). So unlike
  shadcn (always vendored), Astryx is a **normal versioned dependency** you
  `upgrade` via codemods — and can selectively eject only components you must
  fork.
- **Agent integration.** `npx astryx init --features agents --agent claude`
  generates `CLAUDE.md` context (component index + rules + CLI reference) pulled
  from the installed version; re-run after upgrades (**VERIFIED**
  `working-with-ai.doc.mjs`).
- **MCP server — real, hosted, read-only docs.** **VERIFIED**
  (`apps/docsite/src/app/mcp/route.ts`): an MCP server over Streamable HTTP
  (via `mcp-handler`) exposing **two tools** — `search(query)` to discover
  components/topics/templates, and `get(name)` for full docs+examples (e.g.
  `get("Table")`, `get("theme", {section: "defineTheme"})`). It is a
  documentation oracle for coding agents, **not** a scaffolder and it writes no
  code. **Useful to us:** yes, during development it lets this agent pull correct
  props/examples instead of guessing; it does not change the build. (Exact
  public endpoint URL/config JSON not captured here — get it from
  `astryx.atmeta.com` MCP docs or `npx astryx` before wiring it into
  `.mcp.json`.)

## (f) Maturity / risk verdict + fallback

**VERIFIED signals (all `gh api`, 2026-07-21):**

| Signal | Value |
|---|---|
| Core version | 0.1.7 (pre-1.0) |
| First release | v0.1.1, 2026-06-26 |
| Release cadence | ~weekly: 0.1.1→0.1.7 over 2026-06-26 … 07-21 |
| Repo created | 2026-01-09 |
| Stars / forks | 9,686 / 716 |
| Open issues+PRs | 272 |
| Last push | 2026-07-21 (daily activity) |
| License | MIT |
| Charts | canary-only, no stable release |

**Reading it honestly.** This is a **~4-week-old public, pre-1.0 (0.1.x)**
release from Meta with heavy activity, real docs, its own test suite, codemod-
based upgrades, and Meta's internal track record behind it. Strengths that
matter for a console: an exceptionally complete Table, real theming, first-class
Vite support without a build plugin, and a normal-dependency model with codemods
(lower long-term maintenance than shadcn's frozen copies). Risks: **0.1.x means
breaking changes are expected** (the `upgrade` codemods exist precisely because
of this); charts are not production-ready; API names are still settling
(`Theme` vs `XDSTheme` both appear in current docs); 272 open items on a young
repo.

**vs shadcn/ui (what it replaces).** shadcn is battle-tested, framework-agnostic
(works with any Tailwind React app), and vendored-by-design (zero upstream
breakage — the code is yours, which is also its cost: no upgrades). Astryx is
younger and StyleX-based but ships far more out of the box (a real data grid,
app shell, HCT theming, agent tooling) and upgrades via codemods. Lock-in:
Astryx couples you to `@astryxdesign/*` + React 19 + StyleX conventions; shadcn
couples you to Tailwind. Astryx's `className`/`swizzle` escape hatches keep the
lock-in soft.

**Go / no-go:** **Lean GO, scoped.** *(This is a lean, not a verified
production endorsement — it rests on repo/doc inspection, not on a console built
and shipped on Astryx.)* Build the console's structural and forms/table surface
on Astryx Path A now: the Table alone likely removes TanStack Table, theming and
dark mode are solid, and the static-asset/Go-embed path is unaffected. **Carve
out charts** to a stable companion (**Recharts**) until `@astryxdesign/charts`
ships a `latest` release. Pin exact `@astryxdesign/*` versions and adopt
`npx astryx upgrade` as the update ritual to absorb 0.1.x churn.

**Fallback if Astryx proves too unstable:** because Path A uses Astryx purely as
a React component library behind `className`/token overrides (no StyleX in *our*
code, no Babel/PostCSS coupling), retreating to the original **shadcn/ui +
Tailwind** plan is a UI-layer swap, not an architecture change — the Vite build,
Go embedding, TanStack Query/Router, and Vitest setup all survive intact. That
reversibility is the main reason the GO is low-risk.

## Open items to verify before committing

1. **UNVERIFIED** — Astryx Table plugins with *server-driven* pagination/sort
   wired to TanStack Query. Check: does `useTablePagination`/`useTableSortable`
   support manual/controlled mode? If not, keep TanStack Table for server data.
2. **UNVERIFIED** — Fresh Path-A Vitest run against `@astryxdesign/core` in a
   Vite SPA (render + role query). Low risk; see §d.
3. **UNVERIFIED** — `vite build` `dist/` size with the full pre-built stylesheet
   vs Path B tree-shaken CSS; confirm bundle is acceptable for `go:embed`.
4. **UNVERIFIED** — Exact MCP endpoint URL + client config for `.mcp.json`.
5. **UNVERIFIED** — `FileInput` drag-drop/progress depth for file-upload surface.

## Sources

Primary (fetched/read): `https://astryx.atmeta.com/`,
`https://astryx.atmeta.com/docs/getting-started`; `gh api repos/facebook/astryx`
(+ `/releases`, `/contents/*`); raw files `packages/core/README.md`,
`packages/core/package.json`, `apps/example-vite/{vite.config.ts,package.json}`,
`packages/build/README.md`, `packages/charts/README.md`,
`packages/themes/neutral/README.md`, `packages/core/src/Table/*`,
`packages/core/src/theme/MediaTheme.doc.mjs`,
`packages/cli/docs/{theme.doc.dense.mjs,working-with-ai.doc.mjs}`,
`apps/docsite/src/app/mcp/route.ts`.
StyleX: [Vite+React install](https://stylexjs.com/docs/learn/installation/vite/vite-react),
Context7 `/facebook/stylex` babel-plugin `test` option.
Secondary/marketing (labeled as claims):
[MarkTechPost](https://www.marktechpost.com/2026/06/27/metas-astryx-brings-a-cli-and-mcp-server-to-an-open-source-react-design-system-agents-can-read/).
