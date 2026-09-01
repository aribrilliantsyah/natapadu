import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import { writeFileSync, mkdirSync } from 'node:fs'
import { resolve } from 'node:path'

// main.go memakai //go:embed all:frontend/dist, yang gagal dikompilasi bila
// direktori itu kosong melompong. Vite mengosongkan dist/ pada setiap build
// (emptyOutDir), sehingga penanda yang dipasang manual selalu ikut terhapus —
// dan pernah membuat CI gagal karena penghapusannya ikut ter-commit.
// Plugin ini menuliskannya kembali setiap selesai build.
function keepDistDirectory() {
  return {
    name: 'keep-dist-directory',
    closeBundle() {
      const dir = resolve(__dirname, 'dist')
      mkdirSync(dir, { recursive: true })
      writeFileSync(resolve(dir, '.gitkeep'), '')
    },
  }
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    svelte(),
    tailwindcss(),
    keepDistDirectory()
  ]
})
