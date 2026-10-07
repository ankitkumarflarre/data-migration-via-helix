// Types mirror the Go JSON of internal/plan, internal/execute and internal/api.

export interface Target { variant: string; field: string }
export interface Transform { case?: string; map?: Record<string, string> }
export interface FieldType { type: string; kind: string; max_len?: number; enum?: string[] }
export interface Sample { row: number; before: string; after: string }

export interface FieldImpact {
  field: string; type: FieldType; required: boolean;
  source: 'sheet' | 'template' | 'generated' | 'reference' | 'override';
  rule_id?: string; column?: string; header?: string; transform?: Transform;
  template?: string; ref_variant?: string; values: number; errors: number;
  samples?: Sample[]; attention?: string; overridden?: boolean; alternatives?: Target[];
}
export interface VariantImpact {
  variant: string; entity: string; scope: 'job' | 'row'; key?: string;
  records: number; fields: FieldImpact[]; template: boolean;
}
export interface EffectiveRule {
  id: string; report_no: number; excel_column: string; header: string;
  report_status: string; confidence: string; target: Target; alternatives: Target[];
  report_locations: string[]; transform: Transform; attention?: string; note?: string;
  excluded: boolean; overridden: boolean; effective_target: Target; effective_transform: Transform;
}
export interface AttentionItem { id: string; kind: string; title: string; detail: string }
export interface PlanRecord {
  variant: string; entity: string; scope: string; key_field?: string; key_value?: string;
  fields: Record<string, unknown>; sheet_fields: string[]; refs?: Record<string, string>; generated?: string[];
}
export interface Plan {
  plan_hash: string; rule_set: string; rule_set_sha: string; helix_bundle: string;
  rules: EffectiveRule[]; impact: VariantImpact[]; attention: AttentionItem[];
  blocking_count: number; warning_count: number; blocked_rows: number;
  job_records: PlanRecord[]; row_count: number;
}
export interface RuleOverride { exclude?: boolean; target?: Target; map?: Record<string, string> }
export interface Overrides { rules?: Record<string, RuleOverride>; templates?: Record<string, string> }
export interface Action { variant: string; action: string; record_id?: string }
export interface Progress {
  state: string; dry_run: boolean; total: number; done: number; written: number; failed: number;
  blocked: number; counts: Record<string, Record<string, number>>; job_actions: Action[];
  error?: string; started_at: string; finished_at?: string;
}
export interface Approval { plan_hash: string; acknowledged: string[]; row_limit: number; dry_run: boolean; approved_at: string; run_id: string }
export interface Job {
  id: string; file_name: string; file_sha: string; rule_set: string; uploaded_at: string;
  sheet_rows: number; headers: Record<string, string>; overrides: Overrides; plan: Plan;
  approvals: Approval[]; progress?: Progress; status: string; helix_url: string;
}
export interface Issue { row: number; column?: string; rule_id?: string; target?: string; severity: string; message: string }
export interface PlanRow { row: number; policy_number: string; blocked: boolean; records: PlanRecord[] }
export interface RowResult { row: number; policy_number: string; status: string; message?: string; actions?: Action[] }
export interface Page<T> { total: number; offset: number; items: T[] }
export interface Health { helix_url: string; helix_reachable: boolean; helix_error?: string; rule_sets: string[]; ledger_records: number }
export interface RuleSetInfo { name: string; rules: number; sheet: string; source_report: string; sha256: string }
export interface Leaf { variant: string; entity: string; title: string; module: string }
export interface SchemaField { key: string; type: FieldType; required: boolean; write_once: boolean; ref_entity?: string }
export interface SchemaVariant { variant: string; entity: string; fields: SchemaField[] }

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method, headers: {} };
  if (body instanceof FormData) init.body = body;
  else if (body !== undefined) {
    init.body = JSON.stringify(body);
    (init.headers as Record<string, string>)['Content-Type'] = 'application/json';
  }
  const res = await fetch(path, init);
  const text = await res.text();
  let data: unknown = undefined;
  try { data = text ? JSON.parse(text) : undefined; } catch { data = text; }
  if (!res.ok) {
    const msg = (data && typeof data === 'object' && 'error' in data) ? String((data as { error: unknown }).error) : `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return data as T;
}

export const api = {
  health: () => call<Health>('GET', '/api/health'),
  ruleSets: () => call<RuleSetInfo[]>('GET', '/api/rulesets'),
  upload: (file: File, ruleSet: string) => {
    const fd = new FormData();
    fd.append('file', file);
    fd.append('rule_set', ruleSet);
    return call<Job>('POST', '/api/jobs', fd);
  },
  job: (id: string) => call<Job>('GET', `/api/jobs/${id}`),
  issues: (id: string, severity = '', offset = 0, limit = 100) =>
    call<Page<Issue>>('GET', `/api/jobs/${id}/issues?severity=${severity}&offset=${offset}&limit=${limit}`),
  rows: (id: string, offset = 0, limit = 20) => call<Page<PlanRow>>('GET', `/api/jobs/${id}/rows?offset=${offset}&limit=${limit}`),
  setOverrides: (id: string, ov: Overrides) => call<Job>('PUT', `/api/jobs/${id}/overrides`, ov),
  approve: (id: string, body: { plan_hash: string; acknowledged: string[]; row_limit: number; dry_run: boolean }) =>
    call<Job>('POST', `/api/jobs/${id}/approve`, body),
  results: (id: string, status = '', offset = 0, limit = 100) =>
    call<Page<RowResult>>('GET', `/api/jobs/${id}/results?status=${status}&offset=${offset}&limit=${limit}`),
  variants: () => call<{ bundle: string; variants: Leaf[] }>('GET', '/api/schema/variants'),
  variant: (v: string) => call<SchemaVariant>('GET', `/api/schema/variants/${encodeURIComponent(v)}`),
};

export const short = (v: string) => v.replace('.property.', '.prop.').replace('.personal', '.pers.');
export const fmtValue = (v: unknown) => (v === null || v === undefined ? '' : typeof v === 'string' ? v : JSON.stringify(v));
