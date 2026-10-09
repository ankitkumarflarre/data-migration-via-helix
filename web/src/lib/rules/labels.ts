// Plain-language descriptions of mapping rules for the Rules tab.
import type { Condition, FieldSource, FieldType, Target, Transform } from '../api';

// The Helix tables the HO rule set writes. "Duck Creek copy" marks the
// property_us_personal_call tables that copy the Duck Creek XML layout (S16).
const tables: Record<string, { name: string; note?: string }> = {
  'policy.property.us-fl.personal.safepoint': { name: 'Policy', note: 'FL SafePoint' },
  'dwelling.property.us.personal': { name: 'Dwelling', note: 'Duck Creek copy' },
  'dwelling_asset.property.personal': { name: 'Dwelling asset' },
  'wind_mitigation_verification.property.us-fl.personal.safepoint': { name: 'Wind mitigation verification', note: 'FL OIR-B1-1802' },
  'location.property.us.personal': { name: 'Location' },
  'location_address.property.us.personal': { name: 'Location address', note: 'Duck Creek copy' },
  'section_icoverages.property.us.personal': { name: 'Section I coverages', note: 'Duck Creek copy' },
  'line.property.us.personal': { name: 'Line', note: 'Duck Creek copy' },
  loss_ratio_analysis: { name: 'Loss ratio analysis', note: 'portfolio reporting' },
  organization: { name: 'Issuer', note: 'organization' },
  product: { name: 'Product' },
  party: { name: 'Policyholder', note: 'party' },
  product_version: { name: 'Product version' },
  policy_term: { name: 'Policy term' },
  policy_revision: { name: 'Policy revision' },
  contract_wording: { name: 'Contract wording' },
  policy_version: { name: 'Policy version' },
  'coverage.property.us.personal': { name: 'Product coverage' },
  'coverage_instance.property.us.personal': { name: 'Coverage on the policy' },
  'coverage_deductible.property.us-fl.personal.safepoint': { name: 'Deductible', note: 'FL SafePoint' },
  'dwelling_feature.property.personal': { name: 'House feature' },
  premium_transaction: { name: 'Premium transaction' },
  rating_record: { name: 'Rating record' },
  'residual_market_placement.property.us.personal': { name: 'Residual market placement', note: 'Citizens' },
};

const words: Record<string, string> = {
  id: 'ID', oir: 'OIR', b1: 'B1', fbc: 'FBC', ctr: 'CTR', ctrfactor: 'CTR factor', bceg: 'BCEG', hvac: 'HVAC',
  icoverages: 'I coverages', us: 'US', fl: 'FL', zip: 'ZIP',
};

/** "rated_territory" → "Rated territory". */
export function humanize(key: string): string {
  const s = key.split('_').filter(Boolean).map((w) => words[w] ?? w).join(' ');
  return s ? s[0].toUpperCase() + s.slice(1) : key;
}

export function tableName(variant: string): string {
  const t = tables[variant];
  if (t) return t.note ? `${t.name} (${t.note})` : t.name;
  return humanize(variant.split('.')[0]);
}

export function isDuckCreekCopy(variant: string): boolean {
  return tables[variant]?.note === 'Duck Creek copy';
}

/** "Coverage on the policy: Coverage a › Limit amount" — the instance names which of several records. */
export const recordName = (variant: string, instance?: string) => instance ? `${tableName(variant)}: ${humanize(instance)}` : tableName(variant);
export const targetText = (t: Target) => `${recordName(t.variant, t.instance)} › ${humanize(t.field)}`;

export function typeText(t?: FieldType): string {
  if (!t) return '';
  switch (t.kind) {
    case 'integer': return 'whole number';
    case 'number': return 'number';
    case 'boolean': return 'yes / no';
    case 'date': return 'date';
    case 'timestamp': return 'date and time';
    case 'enum': return `one of: ${(t.enum ?? []).join(', ')}`;
    case 'reference': return 'link to another record';
    default: return t.max_len ? `text, up to ${t.max_len} characters` : 'text';
  }
}

export const q = (v: string) => `“${v}”`;

export function colLabel(col: string, headers: Record<string, string>): string {
  return headers[col] ? `${headers[col]} (column ${col})` : `column ${col}`;
}

export function mapText(map?: Record<string, string>): string {
  const e = Object.entries(map ?? {});
  if (!e.length) return '';
  return e.map(([from, to]) => `${q(from)} → ${to === '' ? 'left empty' : to}`).join(', ');
}

/** What happens to the cell value, as short sentences. */
export function transformText(t: Transform): string[] {
  const out: string[] = [];
  if (t.case === 'upper') out.push('Changed to upper case');
  if (t.case === 'lower') out.push('Changed to lower case');
  const m = mapText(t.map);
  if (m) out.push(`Translated: ${m}. Other values are written as they are.`);
  if (!out.length) out.push('Copied as it is');
  return out;
}

export function whenText(w: Condition | undefined | null, headers: Record<string, string>): string {
  if (!w) return '';
  if (w.not_in?.length) return `Only when ${colLabel(w.column, headers)} has a value other than ${w.not_in.map(q).join(' or ')}`;
  return `Only when ${colLabel(w.column, headers)} is ${(w.in ?? []).map(q).join(' or ')}`;
}

export function formatText(f: string, headers: Record<string, string>): string {
  const parts = [...f.matchAll(/\{([A-Z]+)\}/g)].map(([, c]) => (c === 'PN' ? '{PN} = policy number' : `{${c}} = ${colLabel(c, headers)}`));
  return `Built from the pattern ${q(f)}${parts.length ? ` (${[...new Set(parts)].join('; ')})` : ''}`;
}

/** Where a template field's value comes from. */
export function sourceText(src: FieldSource, headers: Record<string, string>): string {
  if (src.const !== undefined) return `Always ${q(src.const)}`;
  if (src.col) {
    const m = mapText(src.map);
    return `Copied from ${colLabel(src.col, headers)}${m ? `, translated: ${m}` : ''}`;
  }
  if (src.format) return formatText(src.format, headers);
  if (src.ref) return `Link to the ${tableName(src.ref)} record`;
  if (src.generate) return 'Generated placeholder: the sheet has no value for it';
  if (src.min_col) return `Earliest ${colLabel(src.min_col, headers)} in the file`;
  return '—';
}

export function when(at: string): string {
  const d = new Date(at);
  return isNaN(d.getTime()) ? at : d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
}
