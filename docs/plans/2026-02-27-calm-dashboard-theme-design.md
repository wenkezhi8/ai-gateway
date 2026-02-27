# Calm Dashboard Theme Design

**Goal:** Add a switchable “calm dashboard” theme (variant + mode) while keeping Apple theme intact, and unify Dashboard/Cache/Routing styling with consistent tokens and Element Plus overrides.

**Style Direction**
- Calm, low-saturation, gray-blue palette
- Default to light mode; dark mode as optional
- Low-contrast surfaces with soft borders and subtle elevation

**Theme Architecture**
- Theme state remains in `useTheme` with `variant` + `mode` persisted to localStorage
- Tokens are expressed as CSS variables under `[data-theme="dashboard"]` and `[data-theme="dashboard"][data-mode="dark"]`
- Pages and Element Plus components consume tokens; no new logic or data flows introduced

**Tokens**
- Core surface + text tokens: `--bg-app`, `--bg-card`, `--border-color`, `--text-primary`, `--text-muted`
- Accent + semantic: `--accent`, `--success`, `--warning`, `--danger`, `--info`
- Component mapping: cards, tables, buttons, inputs, and charts map to the above tokens

**UI Components Impact**
- Element Plus overrides only within theme tokens (no Apple theme changes)
- Dashboard cards use consistent padding, border, and typography hierarchy
- Cache/Routing remove hardcoded colors and rely on tokens

**Accessibility & Readability**
- Maintain readable contrast on text while keeping a soft surface look
- Dark mode uses deep blue-gray background with light text, avoiding pure black

**Testing & Verification**
- Theme logic already covered by unit tests
- Full UI verification via `npm run test:unit`, `npm run typecheck`, `npm run build`
