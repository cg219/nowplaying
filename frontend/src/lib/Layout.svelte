<script lang="ts">
import type { Snippet } from "svelte";

type Link = {
    url: string
    name: string
    current: boolean
}

type Props = {
    title: string
    subtitle: string
    links?: Link[]
    children: Snippet
}

let { title, subtitle, links, children }: Props = $props();
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
                    <a href="/api/logout" class="contrast">Logout</a>
                </li>
            </ul>
        {/if}
    </nav>

    <section>
        {@render children()}
    </section>
</main>

