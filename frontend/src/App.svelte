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
  import { GetCurrentUser, GetSetting } from '../wailsjs/go/main/App';

  // Splash tampil sampai sesi selesai diperiksa, dengan durasi minimum
  // supaya tidak berkedip di mesin cepat.
  let booting = $state(true);

  onMount(async () => {
    const minimum = new Promise(r => setTimeout(r, 1100));
    try {
      // Tema tersimpan di database adalah sumber kebenaran; localStorage hanya cermin
      const saved = await GetSetting('theme', '');
      if (saved === 'light' || saved === 'dark') applyTheme(saved);
    } catch {}
    try {
      const s = await GetCurrentUser();
      if (s?.user) currentUser.set(s);
    } catch {}
    await minimum;
    booting = false;
  });
</script>

{#if booting}
  <SplashScreen />
{/if}

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
