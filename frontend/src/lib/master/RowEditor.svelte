<script lang="ts">
  import { showToast } from '../stores/appState';
  import { X, Plus, Trash2, Copy, ClipboardPaste, AlertCircle } from 'lucide-svelte';
  import { SaveDataRow, SaveDataRows } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let { tpl, rowId = 0, initial = {}, onClose, onSaved }: {
    tpl: models.Template;
    rowId?: number;                          // > 0 = ubah satu baris yang sudah ada
    initial?: Record<string, any>;
    onClose: () => void;
    onSaved: () => void;
  } = $props();

  const editing = rowId > 0;
  const cols = tpl.columns ?? [];

  const blankRow = (): Record<string, string> =>
    Object.fromEntries(cols.map(c => [c.fieldName, c.defaultValue ?? '']));

  const fromInitial = (): Record<string, string> =>
    Object.fromEntries(cols.map(c => [
      c.fieldName,
      initial[c.fieldName] !== null && initial[c.fieldName] !== undefined ? String(initial[c.fieldName]) : '',
    ]));

  // Mode ubah selalu satu baris; mode tambah dimulai dengan tiga baris kosong
  let rows = $state<Record<string, string>[]>(
    editing ? [fromInitial()] : [blankRow(), blankRow(), blankRow()]
  );
  let saving = $state(false);
  // Kunci "indeksBaris:namaKolom" -> alasan, untuk menandai sel yang bermasalah
  let cellErrors = $state<Record<string, string>>({});

  const filledCount = $derived(
    rows.filter(r => cols.some(c => (r[c.fieldName] ?? '').trim() !== '')).length
  );
  const errorCount = $derived(Object.keys(cellErrors).length);

  const inputType = (t: string) =>
    t === 'DATE' ? 'date'
    : t === 'DATETIME' ? 'datetime-local'
    : ['INTEGER','DECIMAL','CURRENCY','PERCENTAGE'].includes(t) ? 'number'
    : 'text';

  function addRow(count = 1) {
    rows = [...rows, ...Array.from({ length: count }, blankRow)];
  }

  function duplicateRow(i: number) {
    rows = [...rows.slice(0, i + 1), { ...rows[i] }, ...rows.slice(i + 1)];
    cellErrors = {};
  }

  function removeRow(i: number) {
    if (rows.length === 1) { rows = [blankRow()]; cellErrors = {}; return; }
    rows = rows.filter((_, idx) => idx !== i);
    cellErrors = {};
  }

  // Tempel dari Excel: baris dipisah newline, kolom dipisah tab.
  // Mulai mengisi dari sel tempat kursor berada, dan menambah baris bila kurang.
  function handlePaste(e: ClipboardEvent, rowIndex: number, colIndex: number) {
    const text = e.clipboardData?.getData('text/plain') ?? '';
    if (!text.includes('\t') && !text.includes('\n')) return; // tempelan biasa, biarkan

    e.preventDefault();
    const grid = text.replace(/\r/g, '').replace(/\n+$/, '').split('\n').map(line => line.split('\t'));

    const next = [...rows];
    while (next.length < rowIndex + grid.length) next.push(blankRow());

    grid.forEach((line, r) => {
      line.forEach((cell, c) => {
        const col = cols[colIndex + c];
        if (col) next[rowIndex + r] = { ...next[rowIndex + r], [col.fieldName]: cell.trim() };
      });
    });

    rows = next;
    cellErrors = {};
    showToast(`${grid.length} baris ditempel`, 'success');
  }

  async function save() {
    saving = true;
    cellErrors = {};
    try {
      if (editing) {
        await SaveDataRow(tpl.id, rowId, rows[0]);
        showToast('Baris diperbarui', 'success');
        onSaved();
        return;
      }

      const res = await SaveDataRows(tpl.id, rows);

      if (res.errors?.length) {
        const marks: Record<string, string> = {};
        for (const e of res.errors) marks[`${e.index}:${e.field}`] = e.reason;
        cellErrors = marks;
        showToast(
          `${res.errors.length} sel perlu diperbaiki — tidak ada baris yang disimpan`,
          'error', 'Validasi gagal', 6000
        );
        return;
      }

      if (res.saved === 0) {
        showToast('Belum ada baris yang diisi', 'warning');
        return;
      }

      showToast(
        `${res.saved.toLocaleString('id-ID')} baris ditambahkan` +
        (res.skipped ? ` (${res.skipped} baris kosong dilewati)` : ''),
        'success'
      );
      onSaved();
    } catch (e: any) {
      showToast(String(e), 'error', 'Gagal menyimpan', 6000);
    } finally { saving = false; }
  }
</script>

<div class="overlay" onclick={e => { if (e.target === e.currentTarget && !saving) onClose(); }}>
  <div class="modal" style="width:min(1100px, 94vw); height:min(680px, 90vh);">
    <div class="modal-hd">
      <div>
        <div class="modal-hd-title">
          {editing ? `Ubah Baris #${rowId}` : 'Tambah Data'}
        </div>
        <div class="modal-hd-sub">
          {editing
            ? `${tpl.name} — divalidasi dengan aturan yang sama seperti import`
            : `${tpl.name} — isi beberapa baris sekaligus, atau tempel langsung dari Excel`}
        </div>
      </div>
      <button class="btn btn-ghost btn-icon btn-xs" disabled={saving} onclick={onClose}><X size={14} /></button>
    </div>

    {#if !editing}
      <div class="topbar" style="height:42px; background:var(--bg-3);">
        <button class="btn btn-outline btn-xs" onclick={() => addRow(1)}>
          <Plus size={12} /> Tambah Baris
        </button>
        <button class="btn btn-ghost btn-xs" onclick={() => addRow(10)}>+10</button>

        <div class="topbar-sep"></div>
        <span style="font-size:11.5px; color:var(--t3); display:flex; align-items:center; gap:5px;">
          <ClipboardPaste size={12} />
          Salin dari Excel lalu tempel di sel mana pun
        </span>

        <div style="flex:1;"></div>
        {#if errorCount > 0}
          <span class="badge badge-red" style="display:inline-flex; align-items:center; gap:4px;">
            <AlertCircle size={11} /> {errorCount} sel bermasalah
          </span>
        {/if}
        <span style="font-size:11.5px; color:var(--t2);">
          {filledCount} dari {rows.length} baris terisi
        </span>
      </div>
    {/if}

    <div style="flex:1; overflow:auto; min-height:0;">
      <table class="tbl" style="min-width:max-content;">
        <thead>
          <tr>
            <th style="width:42px; text-align:right; color:var(--t3);">#</th>
            {#each cols as col}
              <th style="min-width:150px;">
                {col.displayName}
                {#if col.required}<span style="color:var(--red);">*</span>{/if}
                <div class="mono" style="font-size:10px; color:var(--t3); font-weight:400; text-transform:none;">
                  {col.dataType}{col.formatPattern ? ` · ${col.formatPattern}` : ''}
                </div>
              </th>
            {/each}
            {#if !editing}<th style="width:64px;"></th>{/if}
          </tr>
        </thead>
        <tbody>
          {#each rows as row, i}
            <tr>
              <td class="mono" style="text-align:right; color:var(--t3); padding:4px 8px;">{i + 1}</td>

              {#each cols as col, ci}
                {@const err = cellErrors[`${i}:${col.fieldName}`]}
                <td style="padding:3px 4px;">
                  {#if col.dataType === 'BOOLEAN'}
                    <select class="select select-sm" class:cell-error={!!err} title={err ?? ''}
                      bind:value={rows[i][col.fieldName]} style="width:100%;">
                      <option value="">—</option>
                      <option value="1">Ya</option>
                      <option value="0">Tidak</option>
                    </select>
                  {:else}
                    <input class="input input-sm" class:cell-error={!!err} title={err ?? ''}
                      type={inputType(col.dataType)}
                      bind:value={rows[i][col.fieldName]}
                      onpaste={e => handlePaste(e, i, ci)}
                      placeholder={col.formatPattern || ''}
                      style="width:100%;" />
                  {/if}
                </td>
              {/each}

              {#if !editing}
                <td style="padding:3px 6px; white-space:nowrap;">
                  <button class="btn btn-ghost btn-icon btn-xs" title="Gandakan baris" onclick={() => duplicateRow(i)}>
                    <Copy size={12} />
                  </button>
                  <button class="btn btn-ghost btn-icon btn-xs" title="Hapus baris"
                    style="color:var(--red);" onclick={() => removeRow(i)}>
                    <Trash2 size={12} />
                  </button>
                </td>
              {/if}
            </tr>
          {/each}
        </tbody>
      </table>

      {#if errorCount > 0}
        <div class="inset" style="margin:10px 14px; padding:10px 12px;">
          <div style="font-size:12px; font-weight:600; color:var(--red); margin-bottom:6px;">
            Tidak ada baris yang disimpan sampai semuanya benar
          </div>
          <div style="display:flex; flex-direction:column; gap:3px; max-height:120px; overflow-y:auto;">
            {#each Object.entries(cellErrors) as [key, reason]}
              {@const [idx, field] = key.split(':')}
              <div style="font-size:11.5px; color:var(--t2);">
                Baris {Number(idx) + 1} ·
                <span style="color:var(--t1);">{cols.find(c => c.fieldName === field)?.displayName ?? field}</span>
                — {reason}
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <div class="modal-ft">
      <button class="btn btn-ghost" disabled={saving} onclick={onClose}>Batal</button>
      <button class="btn btn-primary" onclick={save} disabled={saving || (!editing && filledCount === 0)}>
        {#if saving}<span class="spin"></span>{/if}
        {editing ? 'Simpan Perubahan' : `Simpan ${filledCount} Baris`}
      </button>
    </div>
  </div>
</div>

<style>
  /* Sel bermasalah ditandai warna DAN judul tooltip, tidak hanya warna */
  .cell-error {
    border-color: var(--red) !important;
    background: var(--red-dim) !important;
  }
</style>
