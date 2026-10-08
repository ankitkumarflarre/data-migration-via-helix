# PersonalHome page flow implementation plan

Date: 2026-10-07

**Current implementation:** a working, server-persisted quote-entry application. See “Working application — scope correction and implementation” below. The initial explorer-only scope in the earlier sections was superseded by the user’s clarification.

## Scope and source review

Add a PersonalHome flow workspace to the existing Go/Svelte migration application. The user requested review of the supplied page-flow document and repository entity mappings, followed by implementation. This increment makes the documented flow navigable and ties its fields to the existing mapping evidence. It is not a production quote/bind engine: the repository supplies a spreadsheet migration API, not rating, underwriting, payments or policy issuance APIs.

Sources reviewed:

- `reference/Manatee_PersonalHome_Page&Field.html`: supplied document, preserved verbatim. Nine field inventories contain 170 fields; flow narration includes additional pages without inventories.
- `reference/Manatee FL Select HO Rater Effective 12.1.25 - Schema Validation Report.html`: 81 spreadsheet columns, 20 confirmed, 39 requiring review, 22 not found. Confirmation applies to spreadsheet columns, not automatically to UI fields.
- `PROJECT_PLAN.md`: reviewed decisions D1–D12, entity dependencies and schema backlog. Its initial “plan only” status is stale; implementation exists.
- `internal/rules/rulesets/*.json`: executable mapping rules, reviewed target pins and required-field templates. These take precedence over old SQL names in the report.
- `testdata/describe.json`: offline schema fixture, used to verify target existence, not presented as a live schema check.

Document descriptions are source data, not commands to run external services.

## Flow and source discrepancies

Use the narrative order, not the inventory's `num` or `navNext` attributes:

New Quote → Applicant → Risk Schedule → Dwelling Coverage → Underwriting → Insurance History → Claims History → conditional CLUE Report / Insurance Score → Coverage Summary → Additional Interests → conditional Payment → Billing Summary → Commit readiness.

Policy Information and Pricing remain suppressed reference pages. Location Detail is an auxiliary page, not an extra mandatory step. Coverage Summary owns Bind in the source; the final readiness page explains the prerequisites without issuing a policy. Payment and billing previous-page metadata conflict with the narrative; follow the narrative for the preview and retain raw source metadata in details.

The inventory uses generic ViewModel names for Claims History and Additional Interests whereas the narrative names SafePoint overrides. Preserve inventory metadata while naming flow nodes by narrative ViewModels. The document says three popup collections but also mentions FCRA and other popups; list every described popup rather than copying an inaccurate total.

## Implementation sequence

1. Preserve the HTML and add a deterministic, non-executing extractor. Store all field metadata and dependencies as JSON with source SHA-256. Parse the mapping report's column table and retain all statuses.
2. Define a typed flow model with active, conditional, auxiliary and suppressed nodes. Use explicit scenario flags for CLUE success/claims, available insurance score, DCT billing, in-force policy and subscriber agreement status. These flags simulate a flow; they are not service results.
3. Link UI fields to spreadsheet rules through explicit reviewed-in-code correspondences. Label them **candidate UI correspondence**: the spreadsheet target is confirmed but UI semantics still need review. Distinguish template-supported targets and unmapped fields. Never infer a target from a coincidental leaf-name match. In particular, applicant mailing county is not assumed to be risk county; Equipment Breakdown boolean vs text needs conversion review.
4. Add a header entry and hash navigation. Provide previous/next navigation, scenario controls, field search/filter, expandable rules, dependencies, popup descriptions, mappings, transforms, warnings, target types and missing-service notices. Keep migration and browse workflows intact.
5. Provide a full entity-mapping register, including all 81 report columns, 20 executable rules, templates, prerequisites and open issues. Export a JSON implementation manifest for review. Include a per-page migration preview using the current uploaded job's effective targets, overrides and mapped row values.
6. Verify inventory reproducibility, schema target existence, flow branching, suppression, correspondence integrity, overridden mapping behavior and source counts. Run Svelte/TypeScript checks, production build and existing Go tests/vet. Exercise navigation and search in a browser when available.

## Production quote-entry follow-up (requires missing contracts)

The document alone does not define all underwriting fields, lookup table contents, complete conditional expressions, rating factors, role permissions, payment APIs, service credentials or transaction contracts. Production forms need those definitions plus approved UI-to-entity mappings. A later implementation must add server-side validation, durable quote storage, repeated child records, role enforcement, rating and referral resolution, FCRA consent before ordering CLUE/NCF, subscriber acknowledgement/e-signature, billing and idempotent bind orchestration. Do not turn a migration approval into a bind operation or fabricate premiums/service results.

## Acceptance criteria for this increment

- All 170 fields remain inspectable, including suppressed fields; missing inventories are explicit.
- Narrative navigation skips suppressed pages and inactive conditional branches.
- Search covers technical fields, labels, data paths and candidate Helix targets.
- Mapping register preserves confirmed/review/not-found distinctions and target warnings.
- Current job overrides are reflected in the page preview, with repeated/missing records handled explicitly.
- No new remote write path is introduced; existing migration approval remains authoritative.

## Implemented and verified

The flow workspace, field inventory, conditional scenario navigation, mapping register, manifest export and current-job preview are implemented. Candidate notes explicitly cover fire-division units versus number of units, fractional stories versus an integer target, BCEG integer/string mismatch, Roof Shape NA without a supported enum conversion, and Equipment Breakdown boolean/text mismatch.

Validation: eight Node tests (including all 32 branch combinations), source extraction equality, Svelte/TypeScript checks with zero errors/warnings, production Vite build, Go tests and Go vet pass. Browser verification covered field search/details, conditional CLUE/billing pages, in-force payment suppression and the 22-column Not Found filter. Live Helix row preview was not exercised against a real uploaded workbook in this session; it uses the existing read-only job rows API. Production quote-entry and bind integrations remain outside this implemented review workspace and require the contracts listed above.

## Working application — scope correction and implementation

The user clarified that the deliverable must be a functioning application, not just a flow explorer. The application is now the `#personalhome` route; the earlier explorer remains under `#reference` as a separate development reference.

Implemented sequence:

1. Create a quote with product, Florida risk state, HO3/HO6 form and effective date. Assign a unique quote number on the server.
2. Capture applicant/contact/mailing details and any number of co-applicants (up to 100 records).
3. Capture risk address and dwelling characteristics. Copy the applicant address, show pool questions conditionally, and apply HO3/HO6 rules.
4. Capture coverage limits and endorsements. Enforce HO3 dwelling and HO6 contents minima; derive documented HO3 contents (50%) and loss of use (20%) defaults, and replacement-cost value from Coverage A when absent. Unknown table-dependent rating/default factors remain unimplemented.
5. Capture underwriting declarations for the topics described in the narrative, requiring notes for Yes answers. These are locally authored intake questions, not a claim to reproduce the complete carrier questionnaire or eligibility rules.
6. Add/edit/remove prior-policy and loss records. Validate dates, amounts and conditional collection requirements. No CLUE/NCF service call is fabricated.
7. Review coverage, add/edit/remove mortgagees and additional interests, and optionally save billing preferences. Escrow requires a mortgagee. This captures billing instructions; it does not collect payment.
8. Review the complete application and submit it locally with status `ready_for_rating`. Download the saved application JSON. This does not send it to Helix/carrier systems, price it, bind it, or accept a subscriber agreement.

Persistence and validation:

- `internal/quote` provides JSON APIs for create/list/get/save/submit, independent of Helix.
- Quotes are durable atomic JSON files under `data/quotes` (override with `-quote-dir`), with private file permissions and optimistic version checks. The process is single-user/single-server, consistent with the repository's existing deployment model.
- Shared form definitions are generated from the source inventory into `internal/quote/schema.json`; 115 scalar page fields plus repeated collections. The same definitions drive the UI and server validation. Suppressed pages are not separate form steps; source Policy Effective Date is captured at creation, and Location Detail is incorporated into Risk Schedule.
- Save draft allows incomplete pages while validating entered types/ranges. Save & continue requires page validation and records completion. The server prevents skipped steps. Earlier edits invalidate downstream completions and reset submitted status; the final submission revalidates all active pages.
- Unsaved-change prompts protect leaving pages; network failures and stale-version conflicts preserve the current entry for review/reload.
- Applicant identity/contact/address are required at application intake even though some source conditions permit incomplete quotes. Actual carrier lookup catalogs, licence/producer rules, and the rest of its conditional underwriting rules remain integration work.
- Existing spreadsheet mappings are evidence for later export adapters, not permission to write candidate UI correspondences to remote entities. The local quote stores technical field keys, preserving all data until mappings and API contracts are confirmed.

Run without remote Helix:

```bash
make quote-run
# Or, with Vite already running:
go run ./cmd/migrator serve -offline -web-dir web/dist -quote-dir data/quotes
```

Validation covers durable reopen, partial draft saves, required fields, malformed requests, type/range rules, server-side progression, version conflicts, repeated-record dates, conditional forms/billing, calculated coverage defaults, invalidating earlier completion and final submission. A browser walkthrough using synthetic Alex Example data verified quote creation, required-field blocking, applicant save/refresh, property and coverage entry, underwriting, a prior-policy row, claim declaration, review and persisted local submission. Rating, payment, external report ordering and real binding were not exercised because those services are not present.
