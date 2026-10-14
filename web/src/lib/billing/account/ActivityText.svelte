<script lang="ts">
 import type {Activity} from '../api';
 import {href,acct} from '../routes';
 let {a,terms}:{a:Activity;terms:{id:string;policy:string}[]}=$props();
 // Amounts link to the payment and policy references to the term, as on the Duck Creek account page.
 const parts=$derived.by(()=>{
  const out:{text:string;link?:string}[]=[];
  const term=terms.find(t=>t.id===a.term_id)??terms.find(t=>a.text.includes(t.policy));
  const re=/(\$[\d,]+\.\d{2})|([A-Z]{4}\d{7}(?: \d{1,2}\/\d{1,2}\/\d{4} - \d{1,2}\/\d{1,2}\/\d{4})?)/g;
  let last=0,m,amountDone=false;
  while((m=re.exec(a.text))){
   if(m.index>last)out.push({text:a.text.slice(last,m.index)});
   if(m[1]&&a.payment_id&&!amountDone){out.push({text:m[1],link:href('payment/'+a.payment_id)});amountDone=true;}
   else if(m[2]&&term&&m[2].startsWith(term.policy)){out.push({text:m[2],link:href(m[2].includes('/')?acct(a.account_number,'term/'+term.id):acct(a.account_number,'term/'+term.id))});}
   else out.push({text:m[0]});
   last=m.index+m[0].length;
  }
  if(last<a.text.length)out.push({text:a.text.slice(last)});
  return out;
 });
</script>
{#each parts as p}{#if p.link}<a href={p.link}>{p.text}</a>{:else}{p.text}{/if}{/each}
