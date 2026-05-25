# wx — Phased Implementation Roadmap

This directory contains the detailed, reviewable plans for evolving `wx` across three progressive phases.

**Core Principles (apply to all phases)**
- NWS-first. The user should get excellent results with zero configuration.
- When NWS cannot provide a capability, only consider public, free, keyless sources (or pure local calculation).
- Preserve the terminal-native, scriptable, fast, beautiful character of the tool.
- Every phase must leave the codebase in a healthier, more maintainable state than it started.

---

## Phase Summary

| Phase | Focus | Key Deliverables | Prerequisite |
|-------|-------|------------------|--------------|
| **1. Foundation** (Tier 1) | Complete the core data story + scripting ergonomics | Hourly forecast, richer quantitative periods (PoP, dewpoint, etc.), `--short` + full templating, sunrise/sunset (pure calc), location favorites + recents in config, alert polish, version fix | Current main |
| **2. Extensions** (Tier 2) | Personalization, freshness, power-user polish | Per-location config (units, radar prefs), first-class FeelsLike, excellent age/freshness UX, exit codes, monitor picker, radar `--save` + prefs | Phase 1 |
| **3. Ambition** (Tier 3) | Long-term extensibility & depth | Provider strategy + design, possible history/climate or nowcast work, library packaging, sustainable architecture | Phase 1 + 2 |

---

## Documents

- **[Phase 1: Foundation](./phase-1-foundation.md)** — The most important phase. Delivers the majority of user-visible value and the architectural base for everything after.
- **[Phase 2: Extensions](./phase-2-extensions.md)** — Builds directly on Phase 1 structures (especially config and output).
- **[Phase 3: Ambition](./phase-3-ambition.md)** — Deliberately larger scope. Includes guardrails and decision points.

---

## How to Use These Documents

1. Read Phase 1 first (it is the most detailed).
2. Review the incremental changes described in Phase 2 and 3.
3. Provide feedback on:
   - Scope cuts or additions
   - Preferred options (e.g., astronomy package location, template syntax, history approach)
   - Sequencing or effort concerns
   - Anything that violates the NWS-first / zero-config rule in your view

Once you approve or give direction on a phase (or specific features), the next step is usually:
- A more granular task list with file-level estimates, or
- A design spike on a controversial area (astronomy algorithm, templating surface, provider interface v2), or
- Direct implementation of early milestones.

---

## Current Status

- [x] Phase 1 detailed plan written
- [x] Phase 2 detailed plan written
- [x] Phase 3 detailed plan written
- [x] This roadmap index created

**Next:** Awaiting your review and direction.

---

*All plans were generated after thorough exploration of the current codebase (models, providers, output, config, radar, monitor, CLI) and relevant NWS API realities.*