<script lang="ts">
  import { currentUser, showToast } from '../stores/appState';
  import { ShieldCheck } from 'lucide-svelte';
  import { Login } from '../../../wailsjs/go/main/App';

  let user = $state('admin'), pass = $state('admin123');
  let loading = $state(false), err = $state('');

  async function submit(e: Event) {
    e.preventDefault();
    if (!user || !pass) return;
    loading = true; err = '';
    try {
      const s = await Login(user, pass);
      currentUser.set(s);
    } catch (e: any) {
      err = String(e).replace(/^Error: /, '');
    } finally { loading = false; }
  }
</script>

<div class="login-wrap">
  <div class="login-card">
    <div class="login-card-top">
      <div class="login-brand-icon">N</div>
      <div class="login-title">Natapadu</div>
      <div class="login-sub">Navigasi Master Data dan Alat Terpadu</div>
    </div>

    <form class="login-card-body" onsubmit={submit}>
      <div class="field">
        <label class="field-label">Username</label>
        <input class="input" type="text" bind:value={user} placeholder="Masukkan username" autocomplete="username" />
      </div>
      <div class="field">
        <label class="field-label">Kata Sandi</label>
        <input class="input" type="password" bind:value={pass} placeholder="Masukkan password" autocomplete="current-password" />
      </div>

      {#if err}
        <div style="padding:8px 10px; border-radius:7px; background:var(--red-dim); border:1px solid rgba(248,113,113,0.25); color:var(--red); font-size:12.5px;">
          {err}
        </div>
      {/if}

      <button type="submit" class="btn btn-primary" style="width:100%; justify-content:center; height:34px;" disabled={loading}>
        {#if loading}<span class="spin"></span>{/if}
        {loading ? 'Memverifikasi...' : 'Masuk ke Sistem'}
      </button>
    </form>

    <div class="login-card-footer">
      <div style="font-size:11.5px; color:var(--t3); display:flex; align-items:center; justify-content:center; gap:5px;">
        <ShieldCheck size={12} style="color:var(--green);" />
        Offline · Semua data tersimpan lokal
      </div>
      <div style="font-size:11px; color:var(--t3); margin-top:3px; font-family:monospace;">
        admin / admin123
      </div>
    </div>
  </div>
</div>
