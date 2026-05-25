# Phase 3: Ambition — Tier 3 Implementation Plan

**Project:** wx  
**Phase:** 3 of 3  
**Status:** For review  
**Prerequisites:** Phase 1 (Foundation) + Phase 2 (Extensions) completed and stable.

---

## 1. Philosophy for Phase 3

Phase 3 is the only phase where we deliberately consider **expanding scope and long-term architecture**. We only add things that preserve the project's core identity:

- NWS-first, zero user configuration required.
- When NWS cannot provide something, only consider public/free sources that require **no API key and no signup**.
- The tool must still feel fast, scriptable, and terminal-native.
- We are willing to say "no" or "documented future work" to features that would require accounts, heavy dependencies, or paid services.

---

## 2. Candidate Features (Filtered)

| Feature | Fits "NWS-first + zero config"? | Recommended Treatment in Phase 3 |
|---------|----------------------------------|----------------------------------|
| Formal multi-provider architecture + extension points | Yes (design only; actual second provider may be minimal) | High priority design work |
| Historical observations + "above/below normal" | Partially (needs storage + a clean NOAA normals source) | Investigate; may be scoped or deferred |
| Quantitative nowcasting (short-term precip rate/intensity) | Yes — deeper use of existing MRMS / IEM data paths already touched by radar | Strong candidate |
| Air quality (AQI) | Only if a truly zero-config public US source exists (e.g. AirNow.gov public feeds) | Conditional — research first |
| UV index | Similar constraint as AQI | Conditional |
| Library / pkg surface for reuse | Yes — packaging decision | High value for ecosystem |
| Major dashboard-oriented TUI or "wx dashboard" mode | Yes — evolution of monitor | Optional / stretch |
| Richer automation (`wx query`, expression language) | Yes | Nice-to-have |
| Moon phase, more astronomy | Yes (pure calculation) | Low cost, high delight |

**Explicit non-starters for Phase 3 (unless the rule changes):** anything requiring API keys, commercial weather services, heavy ML models, or persistent cloud accounts.

---

## 3. Major Workstreams

### 3.1 Provider Extensibility & Second-Provider Strategy

**Goals**
- Make the existing registry pattern more intentional and documented.
- Decide on the exact shape of a "v2" provider interface if richer capabilities (hourly quantitative, astronomy, radar) become first-class requirements.
- Ship at minimum a design + skeleton. A real second provider (e.g., a thin Open-Meteo adapter for non-US or graceful degradation) is optional in this phase.

**Key Decisions to Document**
- How does a second provider declare which fields it can supply?
- Fallback policy (NWS primary, secondary only for unsupported locations or missing fields).
- How do models evolve to accommodate providers with different data density?

### 3.2 Historical / Climate Context (Investigation + Possible Implementation)

Possible approaches (in priority order):
1. Purely local history: append-only observation log in `~/.cache/wx/history/` (opt-in).
2. NOAA Climate Normals (public data) for "this day normal" comparisons.
3. Hybrid.

Requires new small storage layer and careful privacy / size considerations. May be delivered as an experimental subcommand (`wx history` or `wx trends`) behind a flag.

### 3.3 Deeper Radar / Nowcast Products

Leverage the mature radar subsystem from Phases 1–2:
- New MRMS-derived products (precip rate, storm total, rotation tracks) if they fit the existing compositing + rendering pipeline.
- Better loop controls or "storm motion" vectors in interactive mode.

### 3.4 Library Packaging

- Introduce `pkg/` (or decide on `github.com/mwirges/wx/models`, `.../provider`, etc.).
- Clear public API surface with godoc examples.
- Internal packages remain `internal/`.
- Goal: other Go programs or alternative frontends can consume the NWS provider cleanly.

### 3.5 Optional: Dashboard / Advanced TUI Exploration

- Evaluate whether the current monitor + radar split should evolve into a more unified "dashboard" model.
- Could include multiple saved locations, sparklines (if we have history), etc.
- This is the highest-risk item for scope creep — only pursue if there is strong desire.

---

## 4. Architectural Work

- Formal **extension points design document** (can live in `docs/`).
- Possible evolution of `WeatherProvider` interface or introduction of optional interfaces (`RadarProvider`, `AstronomyProvider`, `QuantitativeForecastProvider`).
- Storage abstraction if history lands (interface + file-based implementation).
- Versioning story for the on-disk config and cache formats (now that they are richer after Phase 2).

---

## 5. File / Package Impact (Illustrative)

- `docs/provider-design.md` (new design doc)
- `pkg/models/`, `pkg/provider/` (or equivalent) — packaging
- `internal/storage/` (if history)
- New or extended providers under `internal/provider/`
- `cmd/history_cmd.go` or experimental subcommands
- Expanded radar product handling

---

## 6. Implementation Philosophy & Guardrails

- Every new data source must have a written justification against the "NWS first + truly zero config" rule.
- Any storage feature must be optional and clearly documented regarding disk usage and privacy.
- Library surface must not pull CLI dependencies into `pkg/`.
- We are comfortable shipping "design + partial implementation" for the biggest items (provider strategy, history) and finishing in a later cycle.

---

## 7. Risks Specific to Phase 3

- Scope explosion — the most dangerous phase.
- Accidental introduction of hidden dependencies or network calls.
- Breaking the "it just works" feeling that makes the tool special.
- Maintenance burden of history or multiple providers.

**Mitigation:** Ruthless scoping + explicit decision records in `docs/`.

---

## 8. Exit Criteria (High Level)

- A clear, reviewed design exists for how the project will support additional providers long-term.
- Any secondary data sources that ship are public, free, keyless, and justified.
- If history / climate features land, they are opt-in and well-bounded.
- The project has a sustainable public package layout.
- All prior phases remain excellent; nothing regressed.
- Documentation (including this roadmap) is updated to reflect the new long-term vision.

---

## 9. Relationship to Earlier Phases

Phase 3 is only valuable **because** Phases 1 and 2 delivered:
- Clean, rich models
- Flexible output (templating)
- Mature per-location config
- Strong radar and monitor foundations
- Proven NWS provider patterns

Without those, the extensibility work would be premature.

---

**Phase 3 is deliberately the "think bigger, but stay true to the spirit" phase.**

Review notes are especially welcome on:
- Which of the candidate features you actually want to pursue vs. explicitly deprioritize.
- Appetite for any storage or history features.
- Desired library packaging approach.
- Whether a formal design doc for providers should be written early in the phase.

Once you give direction, we can turn the selected items into concrete task lists or spike plans.