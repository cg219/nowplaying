import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { resolve } from "node:path"

export default defineConfig({
  plugins: [svelte()],
    build: {
        rollupOptions: {
            input: {
                auth: resolve(__dirname,  "entrypoints/auth.html"),
                settings: resolve(__dirname,  "entrypoints/settings.html"),
                user: resolve(__dirname,  "entrypoints/user.html"),
                reset: resolve(__dirname,  "entrypoints/reset.html")
            },
            output: {
                dir: "build"
            }
        }
    }
})
