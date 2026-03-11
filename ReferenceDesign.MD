# Reference Image Design for Terraria Sprite Pipeline

## 1. Purpose

This document defines a new feature for the current agent pipeline:

- Detect when a user prompt references an existing, recognizable subject
  (character, person, branded item, franchise object, etc.).
- Fetch visual reference candidates from Google Images.
- Have Architect approve a reference (or retry once).
- Use FLUX Klein image-to-image when a reference is approved.
- Keep the current text-only flow as fallback.

This is a design-only document. No implementation details or code changes are included here.

---

## 2. Problem Statement

Current text-only generation performs well for generic concepts but struggles when the user wants:

- Existing characters
- Recognizable people
- Specific objects from known IPs/shows/games

Without a reference image, the model often misses identity and design language.  
The goal is to provide visual grounding only when needed.

---

## 3. Scope

### In Scope

- Architect decides whether a reference is needed.
- Google Images used to retrieve candidate references.
- Candidate filtering/ranking for:
  - single-subject focus
  - illustration/pixel-art preference
  - clean composition
- Architect approval of one candidate (or one retry).
- Automatic generation-mode switch:
  - `text_to_image` (no approved reference)
  - `image_to_image` (approved reference)

### Out of Scope (for now)

- Licensing/provenance enforcement
- Safety/NSFW moderation
- Production-grade caching/versioning/auditing
- Complex multi-retry orchestration (beyond one retry)
- Advanced image-quality constraints (resolution/aspect hard checks)

---

## 4. High-Level Architecture

## Agents / Responsibilities

1. `Architect`
- Parses user intent.
- Produces standard manifest fields.
- Decides `reference_needed`.
- Defines `reference_subject` query seed.
- Reviews candidate references and approves one or requests retry.

2. `Reference Finder` (new)
- Runs Google Images search using query strategy.
- Returns top candidate set (URLs + metadata).
- Applies lightweight filtering/ranking heuristics.

3. `Pixelsmith`
- Receives final manifest.
- Uses `generation_mode`:
  - no reference -> text-to-image
  - approved reference -> image-to-image
- Still runs the same post-processing pipeline.

---

## 5. Data Contract Changes

The Architect output manifest should add:

- `reference_needed: bool`
- `reference_subject: string | null`
- `reference_image_url: string | null`
- `generation_mode: "text_to_image" | "image_to_image"`

Optional helper fields:

- `reference_attempts: int`
- `reference_notes: string | null`

## Expected behavior

- If `reference_needed=false`:
  - `reference_subject=null`
  - `reference_image_url=null`
  - `generation_mode="text_to_image"`

- If `reference_needed=true` and approval succeeds:
  - `reference_subject` set
  - `reference_image_url` set
  - `generation_mode="image_to_image"`

- If approval fails after one retry:
  - `reference_image_url=null`
  - `generation_mode="text_to_image"`
  - keep normal flow

---

## 6. Decision Policy

## 6.1 Triggering `reference_needed`

Architect sets `reference_needed=true` when prompt strongly implies:

- A named person/character (e.g., "Naruto", "Batman", "Messi")
- A specific recognizable item/weapon/object from known media
- A direct request to mimic an existing look

Architect sets `reference_needed=false` for generic fantasy/custom ideas with no explicit identity dependency.

## 6.2 Confidence Behavior

If uncertain, bias toward `true` only when identity accuracy appears important to user intent.

---

## 7. Reference Retrieval Strategy

Google Images query generation:

1. Base: `"<reference_subject>"`
2. Preferred variants:
   - `"<reference_subject> illustration"`
   - `"<reference_subject> concept art"`
   - `"<reference_subject> pixel art"`
3. Optional disambiguation tokens from prompt:
   - outfit/form/era/version keywords

Candidate pool:

- Retrieve top 8 image candidates.

---

## 8. Candidate Filtering & Ranking

Apply lightweight heuristics (as agreed):

1. Prefer single-subject images.
2. Prefer clean composition (less UI/collage/poster clutter).
3. Prefer style order:
   - pixel art
   - illustration/concept art
   - photo (fallback only)

Output:

- Return top 3 ranked candidates for Architect review.

---

## 9. Approval Loop

1. Architect reviews top 3 candidates.
2. If one is acceptable:
   - set `reference_image_url`
   - proceed with `image_to_image`
3. If none acceptable:
   - refine query once
   - rerun finder
4. If still none acceptable:
   - fallback to `text_to_image`
   - continue pipeline (no hard failure)

Retry limit:

- Exactly 1 retry round.

---

## 10. Generation Mode Behavior

## 10.1 Text-to-Image Mode

Used when:

- `reference_needed=false`, or
- reference approval failed

Behavior:

- Current prompt flow remains unchanged.

## 10.2 Image-to-Image Mode

Used when:

- `reference_image_url` is approved

Behavior:

- Pass reference image to FLUX Klein image-to-image call.
- Prompt remains Terraria sprite-focused.
- Keep existing downscale + outline post-processing.

---

## 11. Pipeline Sequence (Conceptual)

1. User prompt enters Architect.
2. Architect emits draft manifest + reference decision.
3. If `reference_needed=true`:
   - Reference Finder fetches/ranks candidates from Google Images.
   - Architect approves best candidate or requests one retry.
4. Architect finalizes manifest including:
   - `generation_mode`
   - optional `reference_image_url`
5. Pixelsmith generates sprite:
   - `text_to_image` or `image_to_image`
6. Post-processing:
   - background removal
   - downscale to Architect-selected icon sizes
   - outline

---

## 12. Failure Handling

Reference flow should not break the run.

Fallback rules:

- Search failure -> fallback to text-to-image.
- No acceptable candidates -> fallback to text-to-image.
- Image download/read failure -> fallback to text-to-image.
- Keep returning a successful manifest with `generation_mode` set appropriately.

---

## 13. Testing Plan (for future implementation)

1. Trigger tests
- Named character prompt -> `reference_needed=true`
- Generic custom weapon prompt -> `reference_needed=false`

2. Finder behavior tests
- Ranking prefers pixel/illustration over photo
- Top-3 selection stable for fixed mock data

3. Approval loop tests
- Accept first round
- Retry once then accept
- Retry once then fallback

4. Mode-switch tests
- Approved URL -> `image_to_image`
- Missing URL -> `text_to_image`

5. End-to-end integration tests
- Prompt with existing character routes through reference path
- Prompt without known identity skips reference path

---

## 14. Open Design Questions

1. Should Architect approval be deterministic heuristic-only, LLM-judged, or hybrid?
2. How much prompt context should be sent to the Reference Finder query rewriter?
3. Should fallback reason be surfaced in manifest for debugging?
4. Should image-to-image strength parameters be configurable per request?

---

## 15. Recommended Initial Defaults

- Candidate fetch count: 8
- Candidates for Architect review: 3
- Retry rounds: 1
- Fallback on any reference-step failure: yes
- Style ranking priority: pixel art > illustration > photo

---

## 16. Summary

This feature introduces a conditional reference pipeline that improves identity fidelity for existing subjects while preserving the current fast path for generic concepts. It keeps Architect as the decision authority and adds one lightweight reference-search loop with safe fallback to the current generation behavior.

