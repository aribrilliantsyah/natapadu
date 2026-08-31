<script lang="ts">
  import { onMount } from 'svelte';
  import { showToast, theme, applyTheme, type Theme } from '../stores/appState';
  import { HardDrive, UserPlus, Key, Folder, CheckCircle2, Sun, Moon, Heart } from 'lucide-svelte';
  import { GetAllUsers, CreateUser, UpdatePassword, BackupDatabase, SelectDirectory, GetDashboardSummary, SetSetting } from '../../../wailsjs/go/main/App';
  import type { models } from '../../../wailsjs/go/models';

  let tab = $state<'account'|'appearance'|'users'|'backup'|'about'>('account');
  let users = $state<models.User[]>([]);
  let dbInfo = $state<models.AppSummary | null>(null);
  let chgId = $state(''), chgPwd = $state('');
  let nName = $state(''), nUser = $state(''), nPwd = $state(''), nRole = $state('USER');
  let bDir = $state(''), baking = $state(false), bPath = $state('');

  async function load() {
    try {
      users = await GetAllUsers();
      dbInfo = await GetDashboardSummary();
      if (users.length && !chgId) chgId = users[0].id;
    } catch { showToast('Gagal memuat', 'error'); }
  }

  async function changePwd() {
    if (!chgPwd.trim()) { showToast('Password tidak boleh kosong', 'warning'); return; }
    try { await UpdatePassword(chgId, chgPwd); showToast('Password diperbarui', 'success'); chgPwd=''; }
    catch (e: any) { showToast('Gagal: ' + e, 'error'); }
  }

  async function createUser() {
    if (!nName.trim()||!nUser.trim()||!nPwd.trim()) { showToast('Lengkapi semua field', 'warning'); return; }
    try {
      await CreateUser(nUser, nPwd, nName, nRole);
      showToast(`User "${nUser}" dibuat`, 'success');
      nName=nUser=nPwd=''; await load();
    } catch (e: any) { showToast('Gagal: ' + e, 'error'); }
  }

  // Tema disimpan ke database supaya tetap ikut kalau file .db dipindah
  async function setTheme(t: Theme) {
    applyTheme(t);
    try { await SetSetting('theme', t); }
    catch (e: any) { showToast('Tema aktif, tapi gagal disimpan: ' + e, 'warning'); }
  }

  async function pickDir() { const d = await SelectDirectory('Folder Backup'); if(d) bDir=d; }

  async function backup() {
    baking=true; bPath='';
    try { bPath = await BackupDatabase(bDir); showToast('Backup berhasil', 'success'); }
    catch (e: any) { showToast('Gagal backup: ' + e, 'error'); }
    finally { baking=false; }
  }

  onMount(load);
</script>

<div style="display:flex; flex-direction:column; height:100%;">
  <div class="topbar">
    <span class="topbar-title">Pengaturan</span>
    <div class="seg" style="margin-left:auto;">
      <button class="seg-btn {tab==='account'?'on':''}" onclick={() => tab='account'}>Akun</button>
      <button class="seg-btn {tab==='appearance'?'on':''}" onclick={() => tab='appearance'}>Tampilan</button>
      <button class="seg-btn {tab==='users'?'on':''}" onclick={() => tab='users'}>Pengguna</button>
      <button class="seg-btn {tab==='backup'?'on':''}" onclick={() => tab='backup'}>Backup DB</button>
      <button class="seg-btn {tab==='about'?'on':''}" onclick={() => tab='about'}>Tentang</button>
    </div>
  </div>

  <div class="scroll-area" style="padding:16px;">

    {#if tab === 'appearance'}
      <div style="max-width:520px; display:flex; flex-direction:column; gap:14px;">
        <div class="panel" style="padding:16px; display:flex; flex-direction:column; gap:12px;">
          <div style="font-size:13px; font-weight:600; color:var(--t1);">Mode Tampilan</div>
          <hr class="sep" />

          <div style="display:grid; grid-template-columns:1fr 1fr; gap:10px;">
            {#each [
              { v: 'dark',  label: 'Gelap',  icon: Moon, hint: 'Nyaman untuk kerja lama & ruangan redup' },
              { v: 'light', label: 'Terang', icon: Sun,  hint: 'Kontras tinggi, cocok untuk ruangan terang' },
            ] as opt}
              {@const Icon = opt.icon}
              <button type="button" class="theme-card {$theme === opt.v ? 'on' : ''}"
                onclick={() => setTheme(opt.v as Theme)}>
                <div class="theme-preview {opt.v}">
                  <span class="tp-bar"></span>
                  <span class="tp-body"></span>
                </div>
                <div style="display:flex; align-items:center; gap:6px; margin-top:9px;">
                  <Icon size={14} />
                  <span style="font-size:13px; font-weight:600;">{opt.label}</span>
                  {#if $theme === opt.v}
                    <CheckCircle2 size={13} style="margin-left:auto;" />
                  {/if}
                </div>
                <div style="font-size:11px; color:var(--t3); margin-top:4px; line-height:1.45; text-align:left;">
                  {opt.hint}
                </div>
              </button>
            {/each}
          </div>

          <div style="font-size:11.5px; color:var(--t3); line-height:1.5;">
            Pilihan tema tersimpan di database aplikasi, jadi ikut terbawa kalau file
            <span class="mono">natapadu.db</span> dipindah ke komputer lain.
          </div>
        </div>
      </div>

    {:else if tab === 'about'}
      <div style="max-width:560px; display:flex; flex-direction:column; gap:14px;">
        <div class="panel" style="padding:20px; display:flex; flex-direction:column; align-items:center; text-align:center; gap:4px;">
          <svg viewBox="0 0 1024 1024" width="72" height="72" aria-hidden="true">
            <defs>
              <linearGradient id="ab-tool" x1="0.1" y1="0" x2="0.9" y2="1">
                <stop offset="0%" stop-color="#a9b8ff"/>
                <stop offset="100%" stop-color="#4f6ef7"/>
              </linearGradient>
              <mask id="ab-wrench">
                <rect width="1024" height="1024" fill="#000"/>
                <circle cx="512" cy="312" r="156" fill="#fff"/>
                <rect x="450" y="300" width="124" height="470" rx="62" fill="#fff"/>
                <circle cx="512" cy="312" r="80" fill="#000"/>
                <polygon points="512,312 372,96 652,96" fill="#000"/>
              </mask>
            </defs>
            <g fill="#4f6ef7" fill-opacity="0.32">
              <rect x="228" y="640" width="82" height="150" rx="28"/>
              <rect x="342" y="566" width="82" height="224" rx="28"/>
              <rect x="456" y="492" width="82" height="298" rx="28"/>
            </g>
            <g transform="rotate(38 512 470) translate(0 60)">
              <rect width="1024" height="1024" fill="url(#ab-tool)" mask="url(#ab-wrench)"/>
            </g>
          </svg>

          <div style="font-size:19px; font-weight:700; color:var(--t1); letter-spacing:-0.3px; margin-top:8px;">
            Natapadu
          </div>
          <div style="font-size:12.5px; color:var(--t2);">
            Navigasi Master Data dan Alat Terpadu
          </div>
          <div style="font-size:11.5px; color:var(--t3);">
            Aplikasi desktop pengolah master data skala besar · 100% lokal &amp; offline
          </div>
        </div>

        <div class="panel" style="padding:16px; display:flex; flex-direction:column; gap:12px;">
          <div style="font-size:13px; font-weight:600; color:var(--t1);">Dibuat Oleh</div>
          <hr class="sep" />

          <div style="font-size:13.5px; font-weight:600; color:var(--t1);">Ari Ardiansyah</div>

          <div style="font-size:12.5px; color:var(--t2); line-height:1.6;">
            Dibangun bersama
            <span class="credit">Claude Code</span>,
            <span class="credit">9 Router</span>, dan
            <span class="credit">Pi Agent</span>.
          </div>

          <hr class="sep" />

          <div style="display:flex; align-items:center; gap:9px; font-size:12.5px; color:var(--t2); line-height:1.6;">
            <Heart size={15} style="color:var(--red); flex-shrink:0;" />
            <span>
              Dan tentunya dibangun dengan cinta,
              untuk membantu pekerjaan istrinya.
            </span>
          </div>
        </div>

        <div class="panel" style="padding:16px; display:flex; flex-direction:column; gap:9px;">
          <div style="font-size:13px; font-weight:600; color:var(--t1);">Teknologi</div>
          <hr class="sep" />
          <div style="display:flex; flex-wrap:wrap; gap:5px;">
            {#each ['Wails v2', 'Go', 'SQLite (modernc)', 'Svelte 5', 'TypeScript', 'Excelize'] as tech}
              <span class="badge badge-gray">{tech}</span>
            {/each}
          </div>
        </div>
      </div>

    {:else}

    {#if tab === 'account'}
      <div style="max-width:380px; display:flex; flex-direction:column; gap:14px;">
        <div class="panel" style="padding:16px; display:flex; flex-direction:column; gap:12px;">
          <div style="font-size:13px; font-weight:600; color:var(--t1); display:flex; align-items:center; gap:7px;">
            <Key size={14} style="color:var(--t2);" /> Ganti Password
          </div>
          <hr class="sep" />
          <div class="field">
            <label class="field-label">Pengguna</label>
            <select class="select" bind:value={chgId}>
              {#each users as u}<option value={u.id}>{u.displayName} ({u.username})</option>{/each}
            </select>
          </div>
          <div class="field">
            <label class="field-label">Password Baru</label>
            <input class="input" type="password" bind:value={chgPwd} placeholder="Masukkan password baru..." />
          </div>
          <button class="btn btn-primary" onclick={changePwd}>Simpan Password</button>
        </div>
      </div>

    {:else if tab === 'users'}
      <div style="display:flex; flex-direction:column; gap:12px;">
        <!-- Add user form inline -->
        <div class="panel" style="padding:14px; display:flex; flex-direction:column; gap:10px;">
          <div style="font-size:12.5px; font-weight:600; color:var(--t1); display:flex; align-items:center; gap:7px;">
            <UserPlus size={13} style="color:var(--t2);" /> Tambah Pengguna
          </div>
          <hr class="sep" />
          <div style="display:grid; grid-template-columns:1fr 1fr 1fr 100px auto; gap:8px; align-items:end;">
            <div class="field">
              <label class="field-label">Nama Lengkap</label>
              <input class="input input-sm" type="text" bind:value={nName} placeholder="Nama..." />
            </div>
            <div class="field">
              <label class="field-label">Username</label>
              <input class="input input-sm" type="text" bind:value={nUser} placeholder="username" />
            </div>
            <div class="field">
              <label class="field-label">Password</label>
              <input class="input input-sm" type="password" bind:value={nPwd} placeholder="Password..." />
            </div>
            <div class="field">
              <label class="field-label">Role</label>
              <select class="select select-sm" bind:value={nRole}>
                <option value="USER">USER</option>
                <option value="ADMIN">ADMIN</option>
              </select>
            </div>
            <button class="btn btn-primary btn-xs" onclick={createUser}><UserPlus size={12} /> Tambah</button>
          </div>
        </div>

        <!-- User table -->
        <div class="panel" style="overflow:hidden;">
          <table class="tbl">
            <thead>
              <tr>
                <th>Nama Lengkap</th><th>Username</th><th>Role</th><th>Status</th><th>Dibuat</th>
              </tr>
            </thead>
            <tbody>
              {#each users as u}
                <tr>
                  <td style="font-weight:500;">{u.displayName}</td>
                  <td class="mono" style="color:var(--t2);">{u.username}</td>
                  <td><span class="badge {u.role==='ADMIN'?'badge-blue':'badge-gray'}">{u.role}</span></td>
                  <td><span class="badge badge-green">{u.status}</span></td>
                  <td class="mono" style="font-size:11.5px; color:var(--t3);">{new Date(u.createdAt).toLocaleDateString()}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

    {:else}
      <div style="max-width:420px; display:flex; flex-direction:column; gap:12px;">
        <div class="panel" style="padding:16px; display:flex; flex-direction:column; gap:12px;">
          <div style="font-size:13px; font-weight:600; color:var(--t1); display:flex; align-items:center; gap:7px;">
            <HardDrive size={14} style="color:var(--t2);" /> Backup Database SQLite
          </div>
          <hr class="sep" />

          <!-- DB stats -->
          <div class="inset" style="padding:10px 12px; display:grid; grid-template-columns:1fr 1fr; gap:10px; font-size:12.5px;">
            <div>
              <div style="color:var(--t3); font-size:11px; margin-bottom:2px;">Ukuran DB</div>
              <div style="font-weight:600; color:var(--t1);">{dbInfo?.databaseSize||'—'}</div>
            </div>
            <div>
              <div style="color:var(--t3); font-size:11px; margin-bottom:2px;">Total Record</div>
              <div class="mono" style="font-weight:600; color:var(--t1);">{dbInfo?.totalRecords?.toLocaleString()||0}</div>
            </div>
          </div>

          <div class="field">
            <label class="field-label">Lokasi Folder Backup</label>
            <div style="display:flex; gap:6px;">
              <input class="input" type="text" readonly
                value={bDir||'Default (~/Documents/Natapadu_Backups)'} style="flex:1; color:var(--t2);" />
              <button class="btn btn-outline" style="flex-shrink:0;" onclick={pickDir}>
                <Folder size={13} />
              </button>
            </div>
          </div>

          <button class="btn btn-primary" style="justify-content:center;" onclick={backup} disabled={baking}>
            {#if baking}<span class="spin"></span>{/if}
            {baking ? 'Memproses Backup...' : 'Buat Backup Sekarang'}
          </button>

          {#if bPath}
            <div class="inset" style="padding:10px; border-color:rgba(52,211,153,0.25); display:flex; flex-direction:column; gap:5px;">
              <div style="display:flex; align-items:center; gap:6px; color:var(--green); font-size:12.5px; font-weight:600;">
                <CheckCircle2 size={13} /> Backup Berhasil
              </div>
              <div data-sel style="font-family:monospace; font-size:11.5px; color:var(--t2); word-break:break-all;">{bPath}</div>
            </div>
          {/if}
        </div>
      </div>
    {/if}
    {/if}
  </div>
</div>

<style>
  .credit {
    color: var(--t1);
    font-weight: 600;
    border-bottom: 1px dashed var(--line-2);
  }

  .theme-card {
    display: flex; flex-direction: column;
    padding: 11px;
    border-radius: 10px;
    border: 1px solid var(--line);
    background: var(--bg-4);
    color: var(--t2);
    cursor: pointer;
    font-family: inherit;
    text-align: left;
    transition: border-color 120ms ease, background 120ms ease;
  }
  .theme-card:hover:not(.on) { border-color: var(--line-2); color: var(--t1); }
  .theme-card.on {
    border-color: var(--accent);
    background: var(--accent-dim);
    color: var(--accent-text);
  }

  /* Pratinjau mini: selalu memakai warna tetap, bukan token,
     supaya kedua kartu memperlihatkan tampilannya masing-masing */
  .theme-preview {
    height: 54px; border-radius: 7px; overflow: hidden;
    display: flex; flex-direction: column;
    border: 1px solid var(--line-2);
  }
  .theme-preview .tp-bar  { height: 13px; display: block; }
  .theme-preview .tp-body { flex: 1; display: block; }
  .theme-preview.dark  .tp-bar  { background: #111318; }
  .theme-preview.dark  .tp-body { background: #16181f; }
  .theme-preview.light .tp-bar  { background: #f4f5f8; }
  .theme-preview.light .tp-body { background: #ffffff; }
</style>
