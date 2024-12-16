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
    let dailytoptracks: Track[] = $state([])
    let dailytopartists: Artist[] = $state([])
    let weeklytoptracks: Track[] = $state([])
    let weeklytopartists: Artist[] = $state([])

    type LastScrobble = {
        artistName: string
        trackName: string
        timestamp: string
    }

    type Artist = {
        name: string
        plays: number
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
            daily: {
                tracks: Track[]
                artists: Artist[]

            }
            weekly: {
                tracks: Track[]
                artists: Artist[]
            }
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

    async function shareDailyArtists() {
        const res = await fetch("/api/share-top-daily-artists", {
            method: "POST",
            credentials: "same-origin"
        })

        await res.json()
    }

    async function shareDailyTracks() {
        const res = await fetch("/api/share-top-daily-tracks", {
            method: "POST",
            credentials: "same-origin"
        })

        await res.json()
    }

    async function shareWeeklyArtists() {
        const res = await fetch("/api/share-top-weekly-artists", {
            method: "POST",
            credentials: "same-origin"
        })

        await res.json()
    }

    async function shareWeeklyTracks() {
        const res = await fetch("/api/share-top-weekly-tracks", {
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
                dailytoptracks = data.top.daily.tracks
                dailytopartists = data.top.daily.artists
                weeklytoptracks = data.top.daily.tracks
                weeklytopartists = data.top.daily.artists
            })
        })
    }

    $effect(() => {
        const id = setInterval(async () => {
            if (document.hasFocus()) {
                const res = await fetch("/api/last-scrobble", {
                    method: "GET",
                    credentials: "same-origin"
                })

                const data = await res.json() as LastScrobble

                artist = data.artistName
                track = data.trackName
                timestamp = data.timestamp
            }
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
        <div class="grid">
            <div class="container">
                <hgroup>
                    <h2>Top Tracks</h2>
                    <span>from the last 24 hours</span>
                </hgroup>
                <ul>
                    {#each dailytoptracks as {name, track, plays}}
                        <li><strong>{name}</strong> - <strong>{track}</strong> ({plays})</li>
                    {/each}
                </ul>
                <p>
                    <button onclick={shareDailyTracks}>Share Top Tracks on Twitter</button>
                </p>
                <hgroup>
                    <h2>Top Tracks</h2>
                    <span>from the last 7 days</span>
                </hgroup>
                <ul>
                    {#each weeklytoptracks as {name, track, plays}}
                        <li><strong>{name}</strong> - <strong>{track}</strong> ({plays})</li>
                    {/each}
                </ul>
                <p>
                    <button onclick={shareWeeklyTracks}>Share Top Tracks on Twitter</button>
                </p>
            </div>
            <div class="container">
                <hgroup>
                    <h2>Top Artists</h2>
                    <span>from the last 24 hours</span>
                </hgroup>
                <ul>
                    {#each dailytopartists as {name, plays}}
                        <li><strong>{name}</strong> ({plays})</li>
                    {/each}
                </ul>
                <p>
                    <button onclick={shareDailyArtists}>Share Top Artists on Twitter</button>
                </p>
                <hgroup>
                    <h2>Top Artists</h2>
                    <span>from the last 7 days</span>
                </hgroup>
                <ul>
                    {#each weeklytopartists as {name, plays}}
                        <li><strong>{name}</strong> ({plays})</li>
                    {/each}
                </ul>
                <p>
                    <button onclick={shareWeeklyArtists}>Share Top Artists on Twitter</button>
                </p>
            </div>
        </div>
    </Layout>
</div>
