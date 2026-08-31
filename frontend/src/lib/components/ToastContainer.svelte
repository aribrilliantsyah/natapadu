<script lang="ts">
  import { toastList } from '../stores/appState';
  import { CheckCircle2, AlertCircle, AlertTriangle, Info, X } from 'lucide-svelte';

  const cfg = {
    success: { icon: CheckCircle2, color: 'var(--green)' },
    error:   { icon: AlertCircle,  color: 'var(--red)' },
    warning: { icon: AlertTriangle, color: 'var(--amber)' },
    info:    { icon: Info,         color: 'var(--accent)' },
  } as any;

  function rm(id: string) { toastList.update(l => l.filter(t => t.id !== id)); }
</script>

<div class="toast-stack">
  {#each $toastList as t (t.id)}
    {@const { icon: Icon, color } = cfg[t.type] ?? cfg.info}
    <div class="toast">
      <Icon size={15} style="color:{color}; flex-shrink:0; margin-top:1px;" />
      <div class="toast-msg">
        {#if t.title}<strong style="display:block; margin-bottom:1px;">{t.title}</strong>{/if}
        <span style="color:var(--t2);">{t.message}</span>
      </div>
      <button class="toast-close" onclick={() => rm(t.id)}><X size={13} /></button>
    </div>
  {/each}
</div>
