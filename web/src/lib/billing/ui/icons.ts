// Heading icons for sections and side panels, picked from the title.
const map: [RegExp, string][] = [
  [/notification/i, 'bell'], [/task/i, 'check'], [/report/i, 'chart'], [/filter/i, 'filter'], [/find|search/i, 'search'],
  [/recent/i, 'clock'], [/activit/i, 'grid'], [/ledger|payment|payee|payor/i, 'dollar'], [/email/i, 'mail'], [/phone/i, 'phone'],
  [/location|address|place/i, 'pin'], [/agent|name|demographic|relationship|people/i, 'user'], [/schedule|term detail|history/i, 'calendar'],
  [/licen|member|account detail|action|hold/i, 'shield'], [/result|list|batch|suspense|error/i, 'database'], [/detail|option|setting|plan/i, 'rules'], [/allocation|disburse|amount|transfer/i, 'transfer'],
];
export const iconFor = (title: string) => map.find(([re]) => re.test(title))?.[1] ?? 'file';
