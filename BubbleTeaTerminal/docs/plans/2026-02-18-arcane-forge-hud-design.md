# Arcane Forge HUD Design

**Date:** 2026-02-18

## Goal
Elevate the existing Bubble Tea terminal flow into a distinct “Arcane Forge HUD” with stronger visual identity, subtle animation, and improved state readability without changing core navigation.

## Scope
- Preserve existing 5-screen flow (input, mode, wizard, forge, staging)
- Add shared shell rendering (ember strip + state frame)
- Add compact-mode fallback for narrow terminals
- Add forge heat/progress feedback and rotating forge verbs
- Add staged item reveal and simple rune/sigil cues in wizard/staging

## Non-Goals
- No persistence layer
- No mouse interactions
- No additional dependencies beyond current Bubble Tea stack

## UI System
- Semantic colors and styles for calm/charged/hot states
- Shared shell wrapper around per-screen content
- Decorative elements disabled in compact mode

## Behavior Additions
- Tick-driven animation counters
- Heat value constrained to 0..100 while forging
- Staging reveal phases for newly crafted item display

## Error Handling
- Keep existing prompt validation behavior
- Keep Ctrl-C quit behavior from all states

## Testing
- Add tests for heat progression bounds
- Add tests for compact mode threshold from window sizing
- Add tests for reveal phase progression and reset behavior
