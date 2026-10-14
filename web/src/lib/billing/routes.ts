// Hash routes of the billing workspace: #billing/<path>. Each pattern names
// the component that renders it; screens.json maps screenshots to these.

export interface Route { pattern: string; component: string; title: string }

export const routes: Route[] = [
  { pattern: '', component: 'Home', title: 'Home' },
  { pattern: 'payments/search', component: 'PaymentSearch', title: 'Search For A Payment' },
  { pattern: 'payment/:payment', component: 'PaymentDetail', title: 'Payment Details' },
  { pattern: 'payments/enter', component: 'EnterPayment', title: 'Enter A Single Payment' },
  { pattern: 'unidentified', component: 'UnidentifiedList', title: 'Unidentified Payments' },
  { pattern: 'unidentified/:payment', component: 'UnidentifiedDetail', title: 'Unidentified Payment' },
  { pattern: 'accounts/new', component: 'NewAccount', title: 'Create New Account' },
  { pattern: 'parties/search', component: 'PartySearch', title: 'Search for a Person or Place' },
  { pattern: 'statements', component: 'ImportStatements', title: 'Import Agent Statement' },
  { pattern: 'reports', component: 'Reports', title: 'Reporting Home' },
  { pattern: 'reports/:report', component: 'Reports', title: 'Reporting Home' },
  { pattern: 'admin', component: 'Admin', title: 'Administration' },
  { pattern: 'admin/errors', component: 'ErrorQueue', title: 'Error Queue' },
  { pattern: 'admin/holds', component: 'HoldEvents', title: 'Manage Hold Events' },
  { pattern: 'admin/holds/new', component: 'Placeholder', title: 'Create Hold Event' },
  { pattern: 'admin/disbursement', component: 'DisbursementService', title: 'Disbursement Interface Service' },
  { pattern: 'admin/transactions', component: 'AdminTransactions', title: 'Billing Transactions' },
  { pattern: 'admin/settings', component: 'AdminSettings', title: 'Workspace Settings' },
  { pattern: 'batches', component: 'Batches', title: 'Batches Awaiting Deposit' },
  { pattern: 'batches/new', component: 'Batches', title: 'Start A New Payment Batch' },
  { pattern: 'batches/processed', component: 'Batches', title: 'Processed Batches' },
  { pattern: 'account/:account', component: 'AccountSummary', title: 'Account' },
  { pattern: 'account/:account/schedule', component: 'Schedule', title: 'Installment Schedule' },
  { pattern: 'account/:account/settings', component: 'AccountSettings', title: 'Select Billing Account Options' },
  { pattern: 'account/:account/pay', component: 'MakePayment', title: 'Make A Payment' },
  { pattern: 'account/:account/suspense-allocation', component: 'SuspenseAllocation', title: 'Suspense Allocation' },
  { pattern: 'account/:account/payments', component: 'PaymentsList', title: 'Payments List' },
  { pattern: 'account/:account/suspense', component: 'Suspense', title: 'Items In Suspense' },
  { pattern: 'account/:account/disburse', component: 'Disburse', title: 'Disburse Funds' },
  { pattern: 'account/:account/eft', component: 'Eft', title: 'Scheduled EFT Transactions' },
  { pattern: 'account/:account/holds', component: 'Holds', title: 'Hold Activity' },
  { pattern: 'account/:account/holds/:hold', component: 'Holds', title: 'Edit Hold Activity' },
  { pattern: 'account/:account/tasks', component: 'Tasks', title: 'Account Tasks' },
  { pattern: 'account/:account/tasks/new', component: 'Tasks', title: 'New Task' },
  { pattern: 'account/:account/transactions', component: 'AccountTransactions', title: 'Billing Transactions' },
  { pattern: 'account/:account/transactions/misc', component: 'Placeholder', title: 'Add Miscellaneous Billing Item' },
  { pattern: 'account/:account/term/:term', component: 'TermSummary', title: 'Policy Term Summary' },
  { pattern: 'account/:account/term/:term/transfer', component: 'TermForms', title: 'Transfer Policy Term' },
  { pattern: 'account/:account/term/:term/collections', component: 'TermForms', title: 'Transfer To Collections' },
  { pattern: 'account/:account/term/:term/plan', component: 'TermForms', title: 'Term Plan Setup' },
  { pattern: 'account/:account/term/:term/renewal-plan', component: 'TermForms', title: 'Renewal Plan Setup' },
  { pattern: 'account/:account/term/:term/hold', component: 'Holds', title: 'Hold Billing for Policy Term' },
  { pattern: 'account/:account/term/:term/pif', component: 'TermForms', title: 'Paid In Full Discount' },
  { pattern: 'account/:account/term/:term/internal-collections', component: 'Placeholder', title: 'Send To Internal Collections' },
  { pattern: 'party/:party', component: 'PartyDetails', title: 'People & Places Details' },
  { pattern: 'party/:party/name/:mode', component: 'PartyName', title: 'Name & Contact Information' },
  { pattern: 'party/:party/demographics', component: 'Demographics', title: 'Demographics' },
  { pattern: 'party/:party/relationships', component: 'Placeholder', title: 'People & Places Relationships' },
  { pattern: 'party/:party/:kind', component: 'PartyRows', title: 'People & Places' },
];

export interface Match { route: Route; params: Record<string, string>; query: URLSearchParams; path: string }

export function parse(hash: string): Match {
  const raw = hash.startsWith('#billing') ? hash.replace(/^#billing\/?/, '') : '';
  const [path, qs] = raw.split('?');
  const parts = path.split('/').filter(Boolean).map(decodeURIComponent);
  for (const route of routes) {
    const pat = route.pattern.split('/').filter(Boolean);
    if (pat.length !== parts.length) continue;
    const params: Record<string, string> = {};
    if (pat.every((p, i) => p.startsWith(':') ? (params[p.slice(1)] = parts[i], true) : p === parts[i])) return { route, params, query: new URLSearchParams(qs ?? ''), path };
  }
  return { route: { pattern: '*', component: 'NotFound', title: 'Not found' }, params: {}, query: new URLSearchParams(qs ?? ''), path };
}

export function href(path: string): string { return '#billing' + (path ? '/' + path.replace(/^\//, '') : ''); }
export function go(path: string): void { location.hash = href(path); }
export const acct = (n: string, sub = '') => `account/${encodeURIComponent(n)}${sub ? '/' + sub : ''}`;
