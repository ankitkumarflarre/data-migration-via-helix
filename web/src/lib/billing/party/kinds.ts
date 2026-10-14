import { codes } from '../codes';

// The effective-dated party sections and their forms (#094–#114). Option
// lists marked "assumed" are not opened in the walkthrough.
export interface KindField { key: string; label: string; required?: boolean; options?: string[]; type?: 'text' | 'textarea' | 'checkbox'; placeholder?: string }
export interface Kind { title: string; add: string; saved: string; modes: ('correct' | 'replace' | 'remove')[]; fields: KindField[]; columns: [string, (f: Record<string, string>) => string][]; ownDates?: boolean; addressGroups?: boolean }

const address: KindField[] = [
  { key: 'country', label: 'Country', options: ['United States'] },
  { key: 'line1', label: 'Address', required: true }, { key: 'line2', label: 'Address 2' },
  { key: 'city', label: 'City', required: true }, { key: 'state', label: 'State', required: true, options: codes.states },
  { key: 'zip', label: 'Zip Code', required: true }, { key: 'plus4', label: '+ 4' }, { key: 'county', label: 'County' },
  { key: 'comments', label: 'Comments', type: 'textarea' }, { key: 'attention', label: 'Attention' },
];
export const oneLine = (f: Record<string, string>) => [f.line1, f.line2].filter(Boolean).join(' ') + `, ${f.city}, ${f.state}, ${f.zip}${f.plus4 ? '-' + f.plus4 : ''} ${f.country ?? 'United States'}`;
const name = (f: Record<string, string>) => [f.prefix, f.first, f.middle, f.last, f.suffix].filter(Boolean).join(' ');

export const kinds: Record<string, Kind> = {
  address: { title: 'Addresses', add: 'Add an address', saved: 'Saved Addresses', modes: ['correct', 'replace'], addressGroups: true, fields: [{ key: 'kind', label: 'Address Type', required: true, options: codes.assumed.addressKinds }, ...address], columns: [['Address', oneLine]] },
  location: { title: 'Locations', add: 'Add a location', saved: 'Saved Locations', modes: [], fields: address, columns: [['Address', oneLine]] },
  phone: { title: 'Phones', add: 'Add a phone', saved: 'Saved Phone Numbers', modes: [], fields: [{ key: 'country', label: 'Country', options: ['United States'] }, { key: 'type', label: 'Phone Type', required: true, options: codes.assumed.phoneTypes }, { key: 'number', label: 'Phone Number', required: true, placeholder: 'XXX-XXX-XXXX' }, { key: 'ext', label: 'Ext' }, { key: 'primary', label: 'Primary Phone', type: 'checkbox' }, { key: 'from', label: 'Contact Hours From', placeholder: 'From' }, { key: 'to', label: 'Contact Hours To', placeholder: 'To' }, { key: 'comments', label: 'Comments', type: 'textarea' }], columns: [['Phone Type', f => f.type], ['Phone Number', f => f.number + (f.ext ? ' x' + f.ext : '')], ['Primary', f => f.primary === 'Yes' ? '✓' : '']] },
  email: { title: 'Email Addresses', add: 'Add an email', saved: 'Saved Email Addresses', modes: ['correct'], fields: [{ key: 'type', label: 'Email Type', required: true, options: codes.assumed.emailTypes }, { key: 'address', label: 'Email Address', required: true }, { key: 'comments', label: 'Comments' }, { key: 'primary', label: 'Primary E-Mail', type: 'checkbox' }], columns: [['Email Type', f => f.type], ['Email Address', f => f.address], ['Comments', f => f.comments ?? ''], ['Primary E-Mail', f => f.primary === 'Yes' ? '✓' : '']] },
  othername: { title: 'Other names', add: 'Add a name', saved: 'Saved Other Names', modes: ['correct', 'replace', 'remove'], fields: [{ key: 'type', label: 'Type of Name', required: true, options: codes.otherNameTypes }, { key: 'prefix', label: 'Prefix', options: codes.assumed.namePrefixes }, { key: 'first', label: 'First' }, { key: 'middle', label: 'Middle' }, { key: 'last', label: 'Last', required: true }, { key: 'suffix', label: 'Suffix', options: codes.assumed.nameSuffixes }, { key: 'comments', label: 'Comments' }], columns: [['Type of Name', f => f.type], ['Name', name], ['Comments', f => f.comments ?? '']] },
  license: { title: 'Licenses', add: 'Add a license', saved: 'Saved Licenses', modes: [], ownDates: true, fields: [{ key: 'type', label: 'License Type', required: true, options: codes.assumed.licenseTypes }, { key: 'number', label: 'License Number', required: true }, { key: 'status', label: 'Status', options: codes.assumed.licenseStatuses }, { key: 'state', label: 'State', options: codes.states }, { key: 'city', label: 'City' }, { key: 'county', label: 'County' }], columns: [['License Type', f => f.type], ['License Number', f => f.number], ['Status', f => f.status ?? ''], ['State', f => f.state ?? '']] },
  membership: { title: 'Memberships', add: 'Add a membership', saved: 'Saved Memberships', modes: [], fields: [{ key: 'type', label: 'Membership Type', required: true, options: codes.membershipTypes }, { key: 'id', label: 'ID', required: true }], columns: [['Membership Type', f => f.type], ['ID', f => f.id]] },
};
