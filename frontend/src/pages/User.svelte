<script lang="ts">
    import Layout from "../lib/Layout.svelte";
    import type { Link } from "../lib/customtypes.ts";
    import type { Action } from "svelte/action";

    let artist: string = $state("")
    let track: string = $state("")
    let timestamp: string = $state("")
    let title: string = $state("")
    let subtitle: string = $state("")
    let links: Link[] = $state([])

    type LastScrobble = {
        artistName: string
        trackName: string
        timestamp: string
    }

    type Props = {
        lastScrobble: LastScrobble
        links: Link[]
        title: string
        subtitle: string
    }

    async function getData() {
        const res = await fetch("/api/me", {
            method: "POST",
            credentials: "same-origin"
        })

        return await res.json() as Props
    }

    const init: Action = () => {
        $effect(() => {
            getData().then((data) => {
                artist = data.lastScrobble.artistName
                track = data.lastScrobble.trackName
                timestamp = data.lastScrobble.timestamp
                title = data.title
                subtitle = data.subtitle
                links = data.links
            })
        })
    }

    $effect(() => {
        const id = setInterval(async () => {
            const res = await fetch("/api/last-scrobble", {
                method: "GET",
                credentials: "same-origin"
            })

            const data = await res.json() as LastScrobble

            artist = data.artistName
            track = data.trackName
            timestamp = data.timestamp
        }, 10 * 1000)

        return () => clearInterval(id)
    })
</script>
<div use:init>
    <Layout title={title} subtitle={subtitle} links={links}>
        <h1>Last Scrobble</h1>
        <div class="last-scrobble">
            <p class="artist">{artist}</p>
            <p class="track">{track}</p>
            <p class="date">{timestamp}</p>
        </div>
    </Layout>
</div>
