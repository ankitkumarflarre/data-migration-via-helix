<script lang="ts">
 // Status shown as a badge: dot + text + tint, never colour alone.
 let {value}:{value:string}=$props();
 const tone=$derived(/past due|error|fail|nsf|reverse|cancel|declin|reject|collection|write/i.test(value)?'bad':/hold|pending|open|suspen|await|queued|due|draft|request/i.test(value)?'warn':/active|paid|deposit|complete|closed|issued|approved|applied|processed|success|current|bill/i.test(value)?'ok':'');
</script>
{#if value}<span class="b-badge {tone}">{value}</span>{/if}
