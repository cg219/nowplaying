import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { resolve } from "@std/path/resolve"

export default defineConfig({
  plugins: [svelte()],
    build: {
        rollupOptions: {
            input: {
                auth: resolve(import.meta.dirname!,  "entrypoints/auth.html"),
                settings: resolve(import.meta.dirname!,  "entrypoints/settings.html"),
                user: resolve(import.meta.dirname!,  "entrypoints/user.html"),
                reset: resolve(import.meta.dirname!,  "entrypoints/reset.html")
            },
            output: {
                dir: "dist"
            }
        }
    }
})
