import './app.css'
import { mount } from 'svelte'
import App from './App.svelte'
import { applyTheme, type Theme } from './lib/stores/appState'

// Terapkan tema sebelum komponen pertama dirender supaya tidak ada kedip gelap.
// Nilai dari SQLite disinkronkan menyusul saat App mount.
try {
  const saved = localStorage.getItem('natapadu:theme')
  applyTheme(saved === 'light' ? 'light' : 'dark')
} catch {
  applyTheme('dark' as Theme)
}

const app = mount(App, {
  target: document.getElementById('app')!,
})

export default app
