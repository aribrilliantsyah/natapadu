import { writable } from 'svelte/store';
import type { models } from '../../../wailsjs/go/models';

export interface ToastMessage {
  id: string;
  type: 'info' | 'success' | 'warning' | 'error';
  title?: string;
  message: string;
  duration?: number;
}

export const currentUser = writable<models.UserSession | null>(null);

// ── Tema ──────────────────────────────────────────────────────────────
// Disimpan di app_settings (SQLite) supaya ikut pindah bersama database,
// dengan cermin di localStorage agar tidak berkedip gelap saat aplikasi dibuka.
export type Theme = 'dark' | 'light';

export const theme = writable<Theme>(readInitialTheme());

function readInitialTheme(): Theme {
  try {
    const saved = localStorage.getItem('natapadu:theme');
    if (saved === 'light' || saved === 'dark') return saved;
  } catch {}
  return 'dark';
}

export function applyTheme(t: Theme) {
  theme.set(t);
  try { localStorage.setItem('natapadu:theme', t); } catch {}
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-theme', t);
  }
}

// Menu utama tunggal — Master Data mengurus navigasinya sendiri (daftar ↔ isi workspace)
export const activeTab = writable<'dashboard' | 'master' | 'history' | 'settings'>('dashboard');

// Workspace yang sedang dibuka. null = tampilkan daftar workspace.
export const openWorkspace = writable<models.Template | null>(null);

// Query aktif di grid workspace — dipakai Export untuk cakupan 'FILTERED' & 'SELECTED'
export interface ActiveQuery {
  templateId: string;
  searchTerm: string;
  filters: models.FilterCondition[];
  filterLogic: 'AND' | 'OR';
  sortBy: string;
  sortOrder: 'ASC' | 'DESC';
  selectedRowIds: number[];
  totalRows: number;
}
export const activeQuery = writable<ActiveQuery>({
  templateId: '', searchTerm: '', filters: [], filterLogic: 'AND', sortBy: '', sortOrder: 'ASC', selectedRowIds: [], totalRows: 0,
});

export const toastList = writable<ToastMessage[]>([]);

export function showToast(message: string, type: 'info' | 'success' | 'warning' | 'error' = 'info', title?: string, duration = 4000) {
  const id = Math.random().toString(36).substring(2, 9);
  toastList.update(items => [...items, { id, type, title, message, duration }]);
  setTimeout(() => {
    toastList.update(items => items.filter(t => t.id !== id));
  }, duration);
}
