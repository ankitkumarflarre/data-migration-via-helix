import targetSchema from './target-schema.json' with { type: 'json' };
import catalog from './catalog.json' with { type: 'json' };
import report from './report.json' with { type: 'json' };
import ruleset from '../../../../internal/rules/rulesets/manatee_fl_select_ho_12_1_25.json' with { type: 'json' };
import templates from '../../../../internal/rules/rulesets/manatee_fl_select_ho_12_1_25.templates.json' with { type: 'json' };

export { catalog, report, ruleset, templates };
export type Field = (typeof catalog.pages)[number]['fields'][number];
export type Inventory = (typeof catalog.pages)[number];
export interface Scenario { clueSuccess: boolean; clueClaims: boolean; scoreAvailable: boolean; billing: boolean; inForce: boolean; agreementPending: boolean }
export const defaultScenario: Scenario = { clueSuccess: false, clueClaims: false, scoreAvailable: false, billing: false, inForce: false, agreementPending: true };
export interface FlowPage { id: string; title: string; vm: string; kind: 'main' | 'conditional' | 'auxiliary' | 'suppressed'; summary: string; gate?: string; gap?: string; inventory?: string }
export const flow: FlowPage[] = [
  { id: 'newquote', title: 'New Quote', vm: 'NewQuote', kind: 'main', summary: 'Select PersonalHome, Florida, product and effective date.', gap: 'Quote creation and product/date eligibility API are not implemented.' },
  { id: 'account', title: 'Applicant', vm: 'Account', kind: 'main', inventory: 'account', summary: 'Insured and co-applicant details, mailing address and producer selection.', gap: 'Party search, producer lookup and Maprisk/PBBI geocoding require service integration.' },
  { id: 'dwellinginfo', title: 'Risk Schedule', vm: 'RiskSchedule', kind: 'main', inventory: 'dwellinginfo', summary: 'HO3/HO6 form, physical characteristics, fire protection, roof and wind mitigation.' },
  { id: 'dwellingcoverage', title: 'Dwelling Coverage', vm: 'DwellingCoverage', kind: 'main', inventory: 'dwellingcoverage', summary: 'Coverage A–F, deductibles, limits and optional endorsements.', gap: 'Coverage defaults and rating lookup tables are referenced but their contents are not supplied.' },
  { id: 'underwriting', title: 'Underwriting', vm: 'Underwriting', kind: 'main', summary: 'FL Select underwriting; alternate FL Advantage, LA and US ViewModels are documented.', gate: 'UnderwritingPage.Show / selected product', gap: 'The source describes question topics but supplies no field inventory or executable decline/referral rules.' },
  { id: 'insurancehistory', title: 'Insurance History', vm: 'InsuranceHistory', kind: 'main', inventory: 'insurancehistory', summary: 'Prior carriers, effective/expiration dates, limits and deductibles.' },
  { id: 'claimshistory', title: 'Claims History', vm: 'ClaimHistorySafePoint', kind: 'main', inventory: 'claimshistory', summary: 'Manual losses, CLUE reports, insurance score and eligibility checks.', gap: 'FCRA consent, CLUE/NCF and eligibility services are not implemented. Flood-loss navigation blocking needs an authoritative rule contract.' },
  { id: 'clue', title: 'CLUE Loss History Report', vm: 'CLUELossHistoryReport', kind: 'conditional', summary: 'Review returned CLUE claims and open claim details.', gate: 'CLUE succeeded AND at least one claim', gap: 'No report field inventory or CLUE service is supplied.' },
  { id: 'score', title: 'Insurance Score Detail', vm: 'NCFDetailPage', kind: 'conditional', summary: 'Insurance score and adverse action reasons.', gate: 'Insurance score available', gap: 'No score field inventory or NCF service is supplied.' },
  { id: 'summary', title: 'Coverage Summary', vm: 'CoverageSafepoint', kind: 'main', summary: 'Rating, premium breakdown, forms, referral/decline messages, subscriber agreement and Bind.', gap: 'Rating, forms, e-signature and binding APIs are not implemented. No premium or eligibility result is fabricated.' },
  { id: 'additionalinterests', title: 'Additional Interests', vm: 'AdditionalOtherInterestsSafePoint', kind: 'main', inventory: 'additionalinterests', summary: 'Mortgagees, additional insureds, interests and premium finance companies.' },
  { id: 'payment', title: 'Make A Payment', vm: 'MakeAPayment', kind: 'conditional', summary: 'Bill class, payment plan, paperless invoice and initial payment.', gate: 'DCT Billing enabled AND policy not in force', gap: 'Payment field inventory and payment gateway are not supplied.' },
  { id: 'billing', title: 'Billing Summary', vm: 'BillingSummary', kind: 'conditional', summary: 'Read-only installment dates and amounts before commitment.', gate: 'DCT Billing enabled (preview); production uses BillingSummaryPage.Show', gap: 'The exact ShowRef rule and billing schedule API are not supplied.' },
  { id: 'commit', title: 'Commit readiness', vm: 'data.CompleteTransactionSetup', kind: 'main', summary: 'Bind originates on Coverage Summary. Commitment produces policy number, documents and downstream notifications.', gap: 'Production binding requires rating, underwriting approval, signed agreement, billing and transaction integrations.' },
  { id: 'locationdetail', title: 'Location Detail', vm: 'LocationDetail', kind: 'auxiliary', inventory: 'locationdetail', summary: 'Location address editing and geocoding, opened from the risk flow.' },
  { id: 'policydetail', title: 'Policy Information', vm: 'PolicyDetail', kind: 'suppressed', inventory: 'policydetail', summary: 'Suppressed by the PersonalHome LOB; retained for field traceability.', gate: 'PolicyDetailPage.Show = False' },
  { id: 'pricing', title: 'Pricing', vm: 'PricingPage', kind: 'suppressed', inventory: 'pricing', summary: 'Suppressed; Coverage Summary provides the active premium page.', gate: 'PricingPage.Show = False' },
];
export function enabled(page: FlowPage, s: Scenario): boolean {
  if (page.kind === 'suppressed' || page.kind === 'auxiliary') return false;
  if (page.id === 'clue') return s.clueSuccess && s.clueClaims;
  if (page.id === 'score') return s.scoreAvailable;
  if (page.id === 'payment') return s.billing && !s.inForce;
  if (page.id === 'billing') return s.billing;
  return true;
}
export function activeFlow(s: Scenario) { return flow.filter(p => enabled(p, s)); }
export function inventoryFor(page: FlowPage) { return catalog.pages.find(p => p.id === page.inventory); }

// Explicit correspondences, never fuzzy field-name inference. Spreadsheet confirmation
// does not establish that a Duck Creek UI value has equivalent semantics/encoding.
export const correspondences: Record<string, string> = {
  'policydetail:2': 'HO-06', 'locationdetail:6': 'HO-03',
  'dwellinginfo:6': 'HO-10', 'dwellinginfo:7': 'HO-11', 'dwellinginfo:8': 'HO-12',
  'dwellinginfo:10': 'HO-14', 'dwellinginfo:18': 'HO-13', 'dwellinginfo:21': 'HO-19',
  'dwellinginfo:23': 'HO-26', 'dwellinginfo:31': 'HO-23', 'dwellinginfo:40': 'HO-25',
  'dwellinginfo:46': 'HO-27', 'dwellinginfo:47': 'HO-28', 'dwellinginfo:48': 'HO-22',
  'dwellingcoverage:35': 'HO-45',
};
export const correspondenceNotes: Record<string, string> = {
  'dwellinginfo:10': 'The UI label says units in fire division; the report separately has unapproved column O for total units within the fire division. Column N is only a candidate and must not be substituted without review.',
  'dwellinginfo:18': 'The UI allows a float while the confirmed Helix target is integer. Fractional stories need a reviewed representation.',
  'dwellinginfo:21': 'The UI declares an integer; the spreadsheet also allows Ungraded and targets a string.',
  'dwellinginfo:31': 'The UI lists NA for masonry/HO6; the Helix enum has hip, gable, flat and other. NA has no approved conversion.',
  'dwellingcoverage:35': 'The UI uses boolean while the target is text. The spreadsheet trim transform is not a boolean adapter.',
};
export const templateLinks: Record<string, {variant: string; field: string; note: string}> = {
  'dwellinginfo:1': { variant: 'policy.property.us-fl.personal.safepoint', field: 'personal_policy_form', note: 'Template column B: HO3 → ho_3, HO6 → ho_6.' },
  'dwellingcoverage:15': { variant: 'policy.property.us-fl.personal.safepoint', field: 'sinkhole_coverage_option', note: 'Template column AP uses Yes/No. UI boolean requires an explicit encoding adapter.' },
};
export interface EffectiveMapping { id: string; effective_target: { variant: string; field: string }; excluded: boolean; overridden: boolean; effective_transform: { case?: string; map?: Record<string, string> } }
export function mappingFor(page: string, field: Field, effective: EffectiveMapping[] = []) {
  const key = `${page}:${field.id}`;
  const rule = ruleset.rules.find(r => r.id === correspondences[key]);
  const current = rule && effective.find(r => r.id === rule.id);
  if (rule) return { status: current?.excluded ? 'Excluded in current job' : 'Candidate UI correspondence', rule, target: current?.effective_target ?? rule.target, transform: current?.effective_transform ?? rule.transform, overridden: current?.overridden ?? false, excluded: current?.excluded ?? false, note: correspondenceNotes[key] ?? 'Spreadsheet mapping is confirmed; UI correspondence needs semantic and value-encoding review.' };
  const template = templateLinks[key];
  if (template) return { status: 'Template correspondence', target: {variant: template.variant, field: template.field}, note: template.note, excluded: false, overridden: false };
  return { status: 'Unmapped', note: 'No approved UI-to-Helix mapping is supplied.', excluded: false, overridden: false };
}
export const popups = [
  { parent: 'account', name: 'Co-applicant / party search', detail: 'Add/remove persons or select an existing party. Party-search service is required.' },
  { parent: 'dwellinginfo', name: 'Location Detail', detail: 'Edit a risk address; OK commits changes and Cancel discards the edit.' },
  { parent: 'claimshistory', name: 'Loss History Detail', detail: 'Add/edit a manual loss. Keep edits isolated until OK; Cancel rolls back.' },
  { parent: 'claimshistory', name: 'CLUE Claim History Detail', detail: 'Separate transaction; OK commits and Cancel rolls back. Agent roles are read-only.' },
  { parent: 'claimshistory', name: 'FCRA / FCRA NCF Disclosure', detail: 'Consent is required before ordering CLUE or insurance score; consent must not be inferred from navigation.' },
  { parent: 'summary', name: 'Subscriber Agreement', detail: 'EmailNotificationACK in SubscriberAgreement; acknowledgement precedes e-signature and is required before Bind.' },
  { parent: 'summary', name: 'Rating Summary', detail: 'Premium Detail opens a peril-level breakdown from the rating engine.' },
  { parent: 'additionalinterests', name: 'AOI Detail', detail: 'Edit a mortgagee/additional insured; Name is required by the LOB override.' },
];
export function matchesField(page: string, field: Field, query: string, effective: EffectiveMapping[] = []) {
  const mapping = mappingFor(page, field, effective);
  return [field.label, field.tech, field.path, field.group, field.validation, field.calc, mapping.target?.variant, mapping.target?.field].some(v => v?.toLowerCase().includes(query.trim().toLowerCase()));
}
export function exportManifest(scenario: Scenario) {
  return { version: 1, scope: 'Documentation and migration review; not a quote or bind request', sources: {catalog: {file: catalog.source, sha256: catalog.sha256}, report: {file: report.source, sha256: report.sha256}}, scenario, activePages: activeFlow(scenario).map(p => p.id), flow, inventory: catalog.pages, correspondences, correspondenceNotes, templateLinks, ruleset, templates, reportColumns: report.columns, popups };
}

export function targetType(target: {variant: string; field: string}) {
  const schema = targetSchema as Record<string, {type: string; required: boolean; enum: string[] | null}>;
  const field = schema[`${target.variant}.${target.field}`];
  return field ? `${field.type}${field.enum?.length && !field.type.startsWith('enum(') ? ` (${field.enum.join(', ')})` : ''}${field.required ? ' · required' : ''}` : 'Not in the offline target snapshot';
}
