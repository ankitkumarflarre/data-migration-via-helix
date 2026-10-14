<script lang="ts">
 let {total,page=$bindable(0),size=10,cols}:{total:number;page?:number;size?:number;cols:number}=$props();
 const pages=$derived(Math.max(1,Math.ceil(total/size)));
</script>
<tfoot><tr><td colspan={cols}><div class="b-pager"><span>{total?`${total} results found. Currently showing ${page*size+1} - ${Math.min(total,(page+1)*size)}.`:'No results to display.'}</span><span class="pg"><button aria-label="Previous page" disabled={page===0} onclick={()=>page=Math.max(0,page-1)}>‹</button><span>Page <b>{total?page+1:0}</b> of {total?pages:0}</span><button aria-label="Next page" disabled={page>=pages-1} onclick={()=>page=Math.min(pages-1,page+1)}>›</button></span></div></td></tr></tfoot>
