import definition from '../../../../internal/quote/schema.json' with {type:'json'};
export type Value = string | number | boolean | null;
export interface Condition {key: string; values: Value[]}
export interface FormField {key:string; label:string; type:string; required:boolean; readOnly?:boolean; default?:Value; options?:string[]; group?:string; hint?:string; min?:number; max?:number; maxLength?:number; showWhen?:Condition; requiredWhen?:Condition}
export interface FormPage {id:string; title:string; fields:FormField[]; notice?:string; when?:Condition; collection?:{key:string;label:string;fields:FormField[];when?:Condition}}
export const pages: FormPage[] = definition.pages;
export interface Quote {id:string;number:string;version:number;status:string;created_at:string;updated_at:string;current_page:string;completed:string[];values:Record<string,Value>;collections:Record<string,Record<string,Value>[]>}
export function matches(c:Condition|undefined,values:Record<string,Value>){return !c || c.values.includes(values[c.key]);}
export const activePages=(values:Record<string,Value>)=>pages.filter(p=>matches(p.when,values));
export class QuoteError extends Error {fields:Record<string,string>;status:number;constructor(message:string,fields:Record<string,string>,status:number){super(message);this.fields=fields;this.status=status;}}
async function call<T>(method:string,path:string,body?:unknown):Promise<T>{
 const response=await fetch('/api/quotes'+path,{method,headers:{'Content-Type':'application/json'},body:body===undefined?undefined:JSON.stringify(body)});
 const text=await response.text();let data;
 try{data=JSON.parse(text);}catch{throw new QuoteError('The quote server is unavailable. Start the Go server and try again.',{},response.status);}
 if(!response.ok)throw new QuoteError(data.error || 'Could not save quote.',data.fields ?? {},response.status);
 return data;
}
export const quotesAPI={list:()=>call<Quote[]>('GET',''),get:(id:string)=>call<Quote>('GET','/'+id),create:(values:Record<string,Value>)=>call<Quote>('POST','',{values}),save:(q:Quote,page:string,values:Record<string,Value>,rows:Record<string,Value>[],advance:boolean)=>call<Quote>('PUT','/'+q.id,{version:q.version,page,values,rows,advance}),submit:(q:Quote)=>call<Quote>('POST','/'+q.id+'/submit',{version:q.version})};
export function pageValues(page:FormPage,values:Record<string,Value>){return Object.fromEntries(page.fields.filter(f=>!f.readOnly).map(f=>[f.key,values[f.key]??null]));}
export function fieldDefaults(fields:FormField[]){return Object.fromEntries(fields.filter(f=>f.default!==undefined).map(f=>[f.key,f.default!]));}
