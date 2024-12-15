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
    let toptracks: Track[] = $state([])
    let topartists: string[] = $state([])

    type LastScrobble = {
        artistName: string
        trackName: string
        timestamp: string
    }

    type Track = {
        name: string
        track: string
        plays: number
    }

    type Props = {
        lastScrobble: LastScrobble
        links: Link[]
        title: string
        subtitle: string
        top: {
            tracks: Track[]
            artists: string[]
        }
    }

    async function getData() {
        const res = await fetch("/api/me", {
            method: "POST",
            credentials: "same-origin"
        })

        return await res.json() as Props
    }

    async function shareLatestTrack() {
        const res = await fetch("/api/share-latest-track", {
            method: "POST",
            credentials: "same-origin"
        })

        await res.json()
    }

    async function shareTopArtists() {
        const res = await fetch("/api/share-top-artists", {
            method: "POST",
            credentials: "same-origin"
        })

        await res.json()
    }

    async function shareTopTracks() {
        const res = await fetch("/api/share-top-tracks", {
            method: "POST",
            credentials: "same-origin"
        })

        await res.json()
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
                toptracks = data.top.tracks
                topartists = data.top.artists
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
        <p>
            <button onclick={shareLatestTrack}>Share Latest on Twitter</button>
        </p>

        <h1>Metrics</h1>

        <hgroup>
            <h2>Top Tracks</h2>
            <span>from the last 7 days</span>
        </hgroup>
        <ul>
            {#each toptracks as {name, track, plays}}
               <li>Plays: {plays} <strong>{name}</strong> - <strong>{track}</strong></li>
            {/each}
        </ul>
        <p>
            <button onclick={shareTopTracks}>Share Top Tracks on Twitter</button>
        </p>

        <hgroup>
            <h2>Top Artists</h2>
            <span>from the last 7 days</span>
        </hgroup>
        <ul>
            {#each topartists as name}
               <li><strong>{name}</strong></li>
            {/each}
        </ul>
        <p>
            <button onclick={shareTopArtists}>Share Top Artists on Twitter</button>
        </p>

    </Layout>
</div>
