<script lang="ts">
  import { showToast } from '../stores/appState';
  import { X } from 'lucide-svelte';
  import { SaveDataRow } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let { tpl, rowId = 0, initial = {}, onClose, onSaved }: {
    tpl: models.Template;
    rowId?: number;                          // 0 = baris baru
    initial?: Record<string, any>;
    onClose: () => void;
    onSaved: () => void;
  } = $props();

  // Semua field dikirim sebagai string — backend yang menjalankan transform + validasi,
  // aturan yang sama persis dengan jalur import.
  let vals = $state<Record<string, string>>(
    Object.fromEntries((tpl.columns ?? []).map(c => [
      c.fieldName,
      initial[c.fieldName] !== null && initial[c.fieldName] !== undefined ? String(initial[c.fieldName]) : (c.defaultValue ?? ''),
    ]))
  );
  let saving = $state(false);

  const inputType = (t: string) =>
    t === 'DATE' ? 'date'
    : t === 'DATETIME' ? 'datetime-local'
    : ['INTEGER','DECIMAL','CURRENCY','PERCENTAGE'].includes(t) ? 'number'
    : 'text';

  async function save() {
    saving = true;
    try {
      await SaveDataRow(tpl.id, rowId, vals);
      showToast(rowId ? 'Baris diperbarui' : 'Baris ditambahkan', 'success');
      onSaved();
    } catch (e: any) {
      showToast(String(e), 'error', 'Validasi gagal', 6000);
    } finally { saving = false; }
  }
</script>

<div class="overlay" onclick={e => { if (e.target === e.currentTarget) onClose(); }}>
  <div class="modal" style="width:560px; max-height:88vh;">
    <div class="modal-hd">
      <div>
        <div class="modal-hd-title">{rowId ? `Ubah Baris #${rowId}` : 'Tambah Baris Manual'}</div>
        <div class="modal-hd-sub">{tpl.name} — divalidasi dengan aturan yang sama seperti import</div>
      </div>
      <button class="btn btn-ghost btn-icon btn-xs" onclick={onClose}><X size={14} /></button>
    </div>

    <div class="modal-body" style="display:flex; flex-direction:column; gap:11px;">
      {#each tpl.columns ?? [] as col}
        <div class="field">
          <label class="field-label">
            {col.displayName}
            {#if col.required}<span style="color:var(--red);">*</span>{/if}
            <span class="mono" style="color:var(--t3); font-weight:400; margin-left:5px; text-transform:none;">
              {col.dataType}{col.formatPattern ? ` · ${col.formatPattern}` : ''}
            </span>
          </label>
          {#if col.dataType === 'BOOLEAN'}
            <select class="select" bind:value={vals[col.fieldName]}>
              <option value="">—</option>
              <option value="1">Ya</option>
              <option value="0">Tidak</option>
            </select>
          {:else}
            <input class="input" type={inputType(col.dataType)} bind:value={vals[col.fieldName]}
              placeholder={col.formatPattern || col.displayName} />
          {/if}
        </div>
      {/each}
    </div>

    <div class="modal-ft">
      <button class="btn btn-ghost" onclick={onClose}>Batal</button>
      <button class="btn btn-primary" onclick={save} disabled={saving}>
        {#if saving}<span class="spin"></span>{/if}
        {rowId ? 'Simpan Perubahan' : 'Tambah Baris'}
      </button>
    </div>
  </div>
</div>
