// Billing API client and the shapes the Go server returns.

export interface Settings { auto_authorize_limit: number; pif?: { discount_basis_points: number; evaluation_days: number } | null; offered_plans: string[]; disbursement_service: { status: string; queued: boolean; queued_at?: string; last_run?: string } }
export interface SystemView { date: string; version: number; settings: Settings; empty: boolean; user: string }
export interface AccountRef { number: string; name: string; address: string; status: string }
export interface Notification { type: string; message: string; received: string }
export interface Task { id: string; title: string; account_number: string; type: string; category: string; due_date: string; description: string; queue: string; assignee?: string; status: string; created: string }
export interface HomeView { recent: AccountRef[]; notifications: Notification[]; tasks_due: Task[]; reports: string[] }
export interface Account { number: string; version: number; status: string; payor_id: string; payee_id: string; target_due_days: number; end_of_month: boolean; account_type: string; account_class: string; collection_method: string; disbursement_hold: string; nsf_counter: number; currency: string; account_plan: string; invoice_due_day?: number; created_on: string; source?: string }
export interface PartyCard { id: string; number: number; name: string; first: string; last: string; line1: string; line2: string; email: string }
export interface AccountLedger { outstanding: number; current_due: number; past_due: number; in_suspense: number; pending_disbursements: number }
export interface TermLedger { outstanding: number; current_due: number; past_due: number; receivable: number; paid: number; written_off: number }
export interface TermRow { id: string; policy: string; reference: string; effective: string; expiration: string; status: string; current_due: number; past_due: number; outstanding: number; on_hold: boolean }
export interface Activity { id: string; account_number: string; at: string; type: string; created_by: string; amount: number | null; text: string; payment_id?: string; term_id?: string }
export interface Tile { invoice_date: string; amount: number; due_date: string }
export interface Allocation { id: string; account_number: string; term_id?: string; direction: string; date: string; status: string; type: string; amount: number; suspense_id?: string }
export interface Identity { last: string; first: string; middle: string; account_ref: string; policy_ref: string; other: string }
export interface Payment { id: string; version: number; account_number?: string; amount: number; currency: string; method: string; entry_date: string; postmark_date?: string; check_number?: string; name_on_check?: string; card_brand?: string; card_last4?: string; name_on_card?: string; fee?: number; transaction_number?: string; reference?: string; status: string; created_by: string; created_at: string; unidentified?: boolean; identity?: Identity; reverse_reason?: string; reversed_on?: string; allocations: Allocation[] }
export interface PaymentView extends Payment { account_name: string; identifier_type: string; identifier: string; system_date: string }
export interface Suspense { id: string; account_number: string; payment_id?: string; payment_source: string; refund_source: string; reference: string; date: string; reason: string; original: number; amount: number; status: string }
export interface Disbursement { id: string; account_number: string; payee_name: string; amount: number; type: string; reason: string; group: string; method: string; method_detail: string; description: string; status: string; request_date: string; user: string; authorized_by?: string; authorized_on?: string; issued_on?: string; payee_changed: boolean }
export interface Hold { id: string; account_number: string; term_id?: string; event_id?: string; type: string; reason: string; start: string; end?: string; status: string; banner: string; term?: string }
export interface HoldEvent { id: string; version: number; name: string; description: string; type: string; reason: string; start: string; end?: string; accounts?: string[] }
export interface ScheduleItem { recv: string; trans: string; amount: number; cash: number; credit: number; write_off: number; redist: number; balance: number }
export interface ScheduleRow { id: string; install_date: string; due_date: string; term: string; term_id: string; invoiced: boolean; items: ScheduleItem[] }
export interface PIFView { configured: boolean; message?: string; potential_discount: number; evaluation_start?: string; evaluation_end?: string }
export interface AccountView { account: Account; name: string; system_date: string; payor: PartyCard | null; payee: PartyCard | null; ledger: AccountLedger; terms: TermRow[]; activities: Activity[]; last_invoice: Tile | null; next_invoice: Tile | null; last_payment: { id: string; date: string; amount: number; method: string } | null; holds: Hold[]; hold_events: HoldEvent[]; suspense: Suspense[]; disbursements: Disbursement[]; tasks: Task[]; payments: Payment[]; schedule: ScheduleRow[]; schedule_history: { date: string; text: string }[]; offered_plans: string[]; auto_authorize_limit: number; pif: Record<string, PIFView> }
export interface Receivable { id: string; date: string; transaction: string; receivable: string; sub_type: string; amount: number; bill_in_full: boolean; effective: string; expiration: string }
export interface Term { id: string; account_number: string; policy_number: string; product: string; effective: string; expiration: string; status: string; payment_plan: string; renewal_plan: string; billing_frequency: string; master_company: string; line_of_business: string; equity_date: string; aggregation_reference: string; potential_commission: number; agent?: { name: string; address: string }; additional_data: { name: string; value: string }[]; receivables: Receivable[] }
export interface TermView { term: Term; account_number: string; account_name: string; account_status: string; version: number; system_date: string; ledger: TermLedger; last_invoice: Tile | null; next_installment: Tile | null; other_terms: TermRow[]; collections_block: string; pif: PIFView; hold: Hold | null; offered_plans: string[]; potential_commission: number }
export interface PartyRow { id: string; effective: string; expiration: string; fields: Record<string, string>; current: boolean }
export interface PartyView { id: string; number: number; version: number; kind: string; associated_with: string; name: string; rows: Record<string, PartyRow[]>; demographics: Record<string, string>; system_date: string; accounts: string[] }
export interface PartyHit { id: string; number: number; name: string; address: string; dob: string; entity_name: string; party_type: string }
export interface Batch { number: string; version: number; entry_date: string; created_at: string; user: string; status: string; control_total: number; batch_total: number; currency: string; default_method: string; org_unit: string; description: string; deposited_on?: string; reversed_on?: string }
export interface ErrorItem { id: string; version: number; at: string; type: string; reference: string; status: string; summary: string; description: string; payload: string; sections: { name: string; value: string }[]; invoice_id: string; async_id: string; attempts: number; resolved_at?: string }
export interface Report { id: string; title: string; date: string; meta: { name: string; value: string }[]; sections: { heading?: string; heading_right?: string; columns: string[]; numeric: boolean[]; rows: string[][]; kinds: string[]; empty?: string }[] }
export interface Receipt { transaction_number: string; base: number; fee: number; total: number; email?: string }
export interface PaymentResult { payment: Payment; message: string; receipt?: Receipt }

export class BillingError extends Error {
  fields: Record<string, string>; status: number; stale: boolean;
  constructor(message: string, fields: Record<string, string>, status: number, stale: boolean) { super(message); this.fields = fields; this.status = status; this.stale = stale; }
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const r = await fetch('/api/billing' + path, { method, headers: { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await r.text();
  let data;
  try { data = JSON.parse(text); } catch { throw new BillingError('The billing server is unavailable. Start the Go server and try again.', {}, r.status, false); }
  if (!r.ok) throw new BillingError(data.error || 'The request failed.', data.fields ?? {}, r.status, !!data.stale);
  return data as T;
}

const enc = encodeURIComponent;
const get = <T>(p: string) => call<T>('GET', p);
const post = <T>(p: string, b: unknown = {}) => call<T>('POST', p, b);
const put = <T>(p: string, b: unknown) => call<T>('PUT', p, b);
type Msg = { message: string };

export const billing = {
  system: () => get<SystemView>('/system'),
  setDate: (version: number, date: string) => put<SystemView>('/system/date', { version, date }),
  runDue: (version: number) => post<{ message: string; invoices: number; past_due: number; issued: number }>('/system/run', { version }),
  saveSettings: (body: { version: number; auto_authorize_limit: number; pif: Settings['pif']; offered_plans: string[] }) => put<SystemView>('/settings', body),
  seed: () => post<SystemView>('/seed'),
  service: (version: number, action: 'start' | 'stop') => post<SystemView>('/disbursement-service', { version, action }),
  home: () => get<HomeView>('/home'),
  findAccounts: (q: string, by = 'Account Number') => get<AccountRef[]>(`/accounts?by=${enc(by)}&q=${enc(q)}`),
  createAccount: (body: { payor_id: string; payee_id: string; same_party: boolean; account_class: string; target_due_days: number; end_of_month: boolean }) => post<Account>('/accounts', body),
  fromQuote: (quote_id: string) => post<Account>('/accounts/from-quote', { quote_id }),
  quoteAccount: (id: string) => get<{ number: string; exists: boolean }>(`/quotes/${id}/account`),
  account: (n: string) => get<AccountView>(`/accounts/${enc(n)}`),
  touch: (n: string) => post<unknown>(`/accounts/${enc(n)}/recent`),
  saveAccountSettings: (n: string, body: { version: number; account_plan: string; invoice_due_day: number }) => put<Msg>(`/accounts/${enc(n)}/settings`, body),
  changeParty: (n: string, body: { version: number; role: string; party_id: string }) => put<Msg>(`/accounts/${enc(n)}/party`, body),
  pay: (n: string, body: unknown) => post<PaymentResult>(`/accounts/${enc(n)}/payments`, body),
  searchPayments: (q: Record<string, string>) => get<PaymentView[]>('/payments?' + new URLSearchParams(q).toString()),
  payment: (id: string) => get<PaymentView>(`/payments/${id}`),
  reverse: (id: string, version: number, reason: string) => post<Payment>(`/payments/${id}/reverse`, { version, reason }),
  unidentified: () => get<PaymentView[]>('/unidentified'),
  enterUnidentified: (body: unknown) => post<Payment>('/unidentified', body),
  updateReference: (id: string, body: unknown) => put<Payment>(`/unidentified/${id}/reference`, body),
  applyUnidentified: (id: string, version: number, account: string) => post<PaymentResult>(`/unidentified/${id}/apply`, { version, account }),
  allocateSuspense: (n: string, body: unknown) => post<Msg>(`/accounts/${enc(n)}/suspense/allocate`, body),
  suspenseAction: (n: string, body: unknown) => post<Msg>(`/accounts/${enc(n)}/suspense/action`, body),
  disburse: (n: string, body: unknown) => post<{ disbursement: Disbursement; message: string }>(`/accounts/${enc(n)}/disbursements`, body),
  decide: (n: string, id: string, version: number, approve: boolean) => post<Disbursement>(`/accounts/${enc(n)}/disbursements/${id}/decision`, { version, approve }),
  placeHold: (n: string, body: unknown) => post<{ hold: Hold; message: string }>(`/accounts/${enc(n)}/holds`, body),
  updateHold: (n: string, id: string, body: unknown) => put<Msg>(`/accounts/${enc(n)}/holds/${id}`, body),
  holdEvents: () => get<HoldEvent[]>('/hold-events'),
  endHoldEvent: (id: string, version: number) => post<Msg>(`/hold-events/${id}/end`, { version }),
  addTask: (n: string, body: unknown) => post<Task>(`/accounts/${enc(n)}/tasks`, body),
  pickup: (n: string, id: string, version: number) => post<Msg>(`/accounts/${enc(n)}/tasks/${id}/pickup`, { version }),
  term: (id: string) => get<TermView>(`/terms/${id}`),
  termPlan: (n: string, id: string, version: number, plan: string) => put<Msg>(`/accounts/${enc(n)}/terms/${id}/plan`, { version, plan }),
  renewalPlan: (n: string, id: string, version: number, plan: string) => put<Msg>(`/accounts/${enc(n)}/terms/${id}/renewal-plan`, { version, plan }),
  transfer: (n: string, id: string, version: number, target: string) => post<Msg>(`/accounts/${enc(n)}/terms/${id}/transfer`, { version, target }),
  collections: (n: string, id: string, version: number) => post<Msg>(`/accounts/${enc(n)}/terms/${id}/collections`, { version }),
  pif: (n: string, id: string, body: unknown) => post<Msg>(`/accounts/${enc(n)}/terms/${id}/pif`, body),
  searchParties: (body: unknown) => post<PartyHit[]>('/parties/search', body),
  createParty: (body: { kind: string; name: Record<string, string>; address: Record<string, string> }) => post<PartyView>('/parties', body),
  party: (id: string) => get<PartyView>(`/parties/${id}`),
  saveRow: (id: string, kind: string, body: unknown) => post<PartyView>(`/parties/${id}/rows/${kind}`, body),
  saveDemographics: (id: string, version: number, fields: Record<string, string>) => put<PartyView>(`/parties/${id}/demographics`, { version, fields }),
  batches: () => get<Batch[]>('/batches'),
  createBatch: (body: unknown) => post<Batch>('/batches', body),
  reverseBatch: (n: string, version: number) => post<Msg>(`/batches/${enc(n)}/reverse`, { version }),
  errors: () => get<ErrorItem[]>('/errors'),
  resubmit: (id: string, body: unknown) => post<ErrorItem>(`/errors/${id}/resubmit`, body),
  statements: () => get<{ id: string; name: string; imported?: string }[]>('/statements'),
  importStatements: (ids: string[]) => post<Msg>('/statements/import', { ids }),
  report: (id: string) => get<Report>(`/reports/${id}`),
};

export function errorText(e: unknown): string { return e instanceof Error ? e.message : String(e); }
export function fieldErrors(e: unknown): Record<string, string> { return e instanceof BillingError ? e.fields : {}; }
