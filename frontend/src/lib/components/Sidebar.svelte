<script lang="ts">
  import { activeTab, currentUser } from '../stores/appState';
  import {
    LayoutDashboard,
    Database,
    History,
    Settings,
    LogOut
  } from 'lucide-svelte';
  import { Logout } from '../../../wailsjs/go/main/App';

  const navItems = [
    { id: 'dashboard', label: 'Dashboard',   icon: LayoutDashboard },
    { id: 'master',    label: 'Master Data', icon: Database },
    { id: 'history',   label: 'Audit & Log', icon: History },
    { id: 'settings',  label: 'Pengaturan',  icon: Settings },
  ] as const;

  async function logout() {
    await Logout();
    currentUser.set(null);
  }
</script>

<aside class="sidebar">
  <!-- Clean Sleek Brand Header -->
  <div class="sidebar-header">
    <div class="sidebar-brand-name">Natapadu</div>
  </div>

  <!-- Nav Menu Items -->
  <nav class="nav">
    {#each navItems as item}
      {@const Icon = item.icon}
      <button
        type="button"
        class="nav-item"
        class:active={$activeTab === item.id}
        onclick={() => activeTab.set(item.id)}
      >
        <span class="nav-icon">
          <Icon size={17} strokeWidth={2} />
        </span>
        <span class="nav-label">{item.label}</span>
      </button>
    {/each}
  </nav>

  <!-- User Status Footer -->
  <div class="sidebar-footer">
    <div class="avatar">
      {$currentUser?.user?.displayName?.[0]?.toUpperCase() ?? 'U'}
    </div>
    <div style="flex:1; min-width:0;">
      <div class="sidebar-user-name">{$currentUser?.user?.displayName || 'Operator'}</div>
      <div class="sidebar-user-role">{$currentUser?.user?.role || 'USER'}</div>
    </div>
    <button
      type="button"
      class="btn btn-ghost btn-icon btn-xs"
      title="Keluar"
      onclick={logout}
      style="color:var(--t3);"
    >
      <LogOut size={14} strokeWidth={2.2} />
    </button>
  </div>
</aside>
