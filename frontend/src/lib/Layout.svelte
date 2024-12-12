<script lang="ts">
import type { Snippet } from "svelte";
import type { Link } from "./customtypes";

type Props = {
    title: string
    subtitle: string
    links?: Link[]
    children: Snippet
}

let { title, subtitle, links, children }: Props = $props();

async function logout(evt: Event) {
    evt.preventDefault()

    const res = await fetch("/auth/logout", {
        method: "POST",
        credentials: "same-origin"
    })

    const data = await res.json()

    if (data.success) location.pathname = "/"
}

</script>

<main class="container">
    <nav role="group">
        <ul>
            <li>
                <hgroup>
                    <h1>{title}</h1>
                    <p>{subtitle}</p>
                </hgroup>
            </li>
        </ul>
        {#if links}
            <ul>
                {#each links as { current, url, name }}
                    {#if current}
                        <li><a href="{url}" aria-current="page">{name}</a></li>
                    {:else}
                        <li><a class="contrast" href="{url}">{name}</a></li>
                    {/if} 
                {/each}
                <li>
                    <a onclick={logout} href="#logout" class="contrast">Logout</a>
                </li>
            </ul>
        {/if}
    </nav>

    <section>
        {@render children()}
    </section>
</main>

