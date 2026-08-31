<script lang="ts">
  import { openWorkspace } from '../stores/appState';
  import type { models } from '../../../wailsjs/go/models';
  import WorkspaceList from './WorkspaceList.svelte';
  import WorkspaceDetail from './WorkspaceDetail.svelte';
  import TemplateDesigner from './TemplateDesigner.svelte';

  let creating = $state(false);
  let startWith = $state<'data' | 'import'>('data');
  let list = $state<WorkspaceList | null>(null);

  function open(tpl: models.Template, mode: 'data' | 'import' = 'data') {
    startWith = mode;
    openWorkspace.set(tpl);
  }
</script>

{#if $openWorkspace}
  {#key $openWorkspace.id}
    <WorkspaceDetail
      tpl={$openWorkspace}
      {startWith}
      onBack={() => { openWorkspace.set(null); list?.reload(); }}
      onChanged={(t) => openWorkspace.set(t)}
    />
  {/key}
{:else}
  <WorkspaceList bind:this={list} onOpen={open} onCreate={() => creating = true} />
{/if}

{#if creating}
  <!-- Workspace baru langsung dibuka supaya alurnya menyambung ke pengisian data -->
  <TemplateDesigner
    source={null}
    onClose={() => creating = false}
    onSaved={(saved) => { creating = false; open(saved); }}
  />
{/if}
