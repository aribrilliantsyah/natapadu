<script lang="ts">
  import { onMount } from 'svelte';
  import { currentUser, activeTab, applyTheme } from './lib/stores/appState';
  import Sidebar from './lib/components/Sidebar.svelte';
  import SplashScreen from './lib/components/SplashScreen.svelte';
  import ToastContainer from './lib/components/ToastContainer.svelte';
  import LoginView from './lib/views/LoginView.svelte';
  import DashboardView from './lib/views/DashboardView.svelte';
  import MasterDataView from './lib/master/MasterDataView.svelte';
  import HistoryView from './lib/views/HistoryView.svelte';
  import SettingsView from './lib/views/SettingsView.svelte';
  import { GetCurrentUser, GetSetting, GetAppVersion, ShowMainWindow } from '../wailsjs/go/main/App';

  // Aplikasi dibuka sebagai kotak splash kecil; jendela baru dibesarkan
  // setelah seluruh persiapan selesai — persis alur aplikasi desktop.
  const MAIN_WINDOW = { width: 1366, height: 860, minWidth: 1024, minHeight: 700 };

  let booting = $state(true);
  let stage = $state('Memulai...');
  let progress = $state(0);
  let version = $state('');

  // Jeda pendek antar tahap: tanpa ini teksnya berganti terlalu cepat untuk dibaca
  const beat = (ms = 180) => new Promise(r => setTimeout(r, ms));

  async function step(label: string, pct: number, work?: () => Promise<unknown>) {
    stage = label;
    progress = pct;
    if (work) { try { await work(); } catch {} }
    await beat();
  }

  onMount(async () => {
    await step('Memuat komponen antarmuka', 15);

    await step('Membaca preferensi tampilan', 40, async () => {
      // Tema tersimpan di database adalah sumber kebenaran; localStorage hanya cermin
      const saved = await GetSetting('theme', '');
      if (saved === 'light' || saved === 'dark') applyTheme(saved);
    });

    await step('Menghubungkan basis data', 65, async () => {
      version = await GetAppVersion();
    });

    await step('Memeriksa sesi pengguna', 88, async () => {
      const s = await GetCurrentUser();
      if (s?.user) currentUser.set(s);
    });

    await step('Siap', 100);

    // Besarkan jendela lebih dulu, baru ganti isinya — urutan sebaliknya
    // membuat tata letak aplikasi sempat terlihat gepeng di ukuran splash.
    try {
      await ShowMainWindow(MAIN_WINDOW.width, MAIN_WINDOW.height, MAIN_WINDOW.minWidth, MAIN_WINDOW.minHeight);
    } catch {}
    await beat(120);
    booting = false;
  });
</script>

{#if booting}
  <SplashScreen {stage} {progress} {version} />
{:else}
  <ToastContainer />

  {#if !$currentUser}
    <LoginView />
  {:else}
    <!-- Desktop window shell -->
    <div style="
      width: 100vw; height: 100vh;
      display: flex;
      background: var(--c-win);
      overflow: hidden;
    ">
      <Sidebar />

      <main style="
        flex: 1;
        min-width: 0;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        background: var(--c-surface);
      ">
        {#if $activeTab === 'dashboard'}
          <DashboardView />
        {:else if $activeTab === 'master'}
          <MasterDataView />
        {:else if $activeTab === 'history'}
          <HistoryView />
        {:else if $activeTab === 'settings'}
          <SettingsView />
        {/if}
      </main>
    </div>
  {/if}
{/if}
