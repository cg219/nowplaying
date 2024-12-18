/// <reference types="svelte" />
/// <reference types="vite/client" />

import { mount } from 'svelte'
import Settings from "./pages/Settings.svelte"
import Reset from "./pages/Reset.svelte"
import User from "./pages/User.svelte"
import Auth from "./pages/Auth.svelte"

const pages = new Map()

pages.set("settings", Settings)
pages.set("reset", Reset)
pages.set("user", User)
pages.set("auth", Auth)

const app = (page: string) => {
    let p = pages.get("*");

    if (pages.has(page)) p = pages.get(page)

    mount(p, {
        target: document.getElementById('app')!,
    })
}

export default app
