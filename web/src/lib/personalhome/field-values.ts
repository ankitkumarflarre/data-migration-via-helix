import type {FormField, Value} from './quote-model';

export function fieldValue(field: FormField, context: Record<string,Value>, rowIndex=0): Value | undefined {
 if (field.unavailable) return undefined;
 if (field.derive==='constant') return field.default;
 if (field.derive==='fullName') return ['FirstName','MiddleName','LastName'].map(k=>context['AccountInput.'+k]).filter(Boolean).join(' ');
 if (field.derive==='coapplicantLabel') return `Co-applicant #${rowIndex+1}`;
 if (field.derive==='rowNumber') return String(rowIndex+1);
 if (field.derive==='interestDescription') return [context.name,context.type].filter(Boolean).join(' · ');
 if (field.derive==='expirationDate') {
  const effective=String(context['PolicyInput.EffectiveDate']??'');
  if (!/^\d{4}-\d{2}-\d{2}$/.test(effective)) return undefined;
  const [year,month,day]=effective.split('-').map(Number);
  const maxDay=new Date(Date.UTC(year+1,month,0)).getUTCDate();
  return `${year+1}-${String(month).padStart(2,'0')}-${String(Math.min(day,maxDay)).padStart(2,'0')}`;
 }
 if (field.derive==='lossOfUseIncluded') {
  const condo=context['DwellingInput.Form']==='HO6';
  const base=context[condo?'CoverageCPersonalPropertyHO46Input.Limit':'CoverageADwellingInput.Limit'];
  return typeof base==='number'?Math.round(base*(condo?.4:.2)):undefined;
 }
 return context[field.key]??field.default;
}
export function mapURL(provider: string, context:Record<string,Value>):string | undefined {
 const address=['Address1','Address2','City','State','ZipCode'].map(k=>context['AccountInput.'+k]).filter(Boolean).join(', ');
 if (!context['AccountInput.Address1'] || !context['AccountInput.City']) return undefined;
 return provider==='bing'?`https://www.bing.com/maps?q=${encodeURIComponent(address)}`:`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address)}`;
}
