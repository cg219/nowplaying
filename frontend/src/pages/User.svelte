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
        image: string
    }

    type Track = {
        name: string
        track: string
        plays: number
        image: string
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
                weeklytoptracks = data.top.weekly.tracks
                weeklytopartists = data.top.weekly.artists
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
        <div class="container">
            <hgroup>
                <h2>Top Tracks</h2>
                <span>from the last 24 hours</span>
            </hgroup>
            <ul>
                {#each dailytoptracks as {name, track, plays, image}, index}
                    <li class="track">
                        <div class="image" style={`anchor-name: --daily-track-anchor-${index}; background-image: url("${image}");`}></div>
                        <p style={`position-anchor: --daily-track-anchor-${index}`}>
                            <strong class="track-name">{track}</strong>
                            <strong class="artist-name">{name}</strong>
                        </p>
                        <span class="plays" style={`position-anchor: --daily-track-anchor-${index}`}>{plays}</span>
                    </li>
                {/each}
            </ul>
            <p>
                <button onclick={shareDailyTracks}>Share Top Tracks on Twitter</button>
            </p>
        </div>
        <div class="container">
            <hgroup>
                <h2>Top Artists</h2>
                <span>from the last 24 hours</span>
            </hgroup>
            <ul>
                {#each dailytopartists as {name, plays, image}, index}
                    <li class="artist">
                        <div class="image" style={`anchor-name: --daily-artist-anchor-${index}; background-image: url("${image}");`}></div>
                        <p style={`position-anchor: --daily-artist-anchor-${index}`}>
                            <strong class="artist-name">{name}</strong>
                        </p>
                        <span class="plays" style={`position-anchor: --daily-artist-anchor-${index}`}>{plays}</span>
                    </li>
                {/each}
            </ul>
            <p>
                <button onclick={shareDailyArtists}>Share Top Artists on Twitter</button>
            </p>
        </div>
        <div class="container">
            <hgroup>
                <h2>Top Tracks</h2>
                <span>from the last 7 days</span>
            </hgroup>
            <ul>
                {#each weeklytoptracks as {name, track, plays, image}, index}
                    <li class="track">
                        <div class="image" style={`anchor-name: --weekly-track-anchor-${index}; background-image: url("${image}");`}></div>
                        <p style={`position-anchor: --weekly-track-anchor-${index}`}>
                            <strong class="track-name">{track}</strong>
                            <strong class="artist-name">{name}</strong>
                        </p>
                        <span class="plays" style={`position-anchor: --weekly-track-anchor-${index}`}>{plays}</span>
                    </li>
                {/each}
            </ul>
            <p>
                <button onclick={shareWeeklyTracks}>Share Top Tracks on Twitter</button>
            </p>
        </div>
        <div class="container">
            <hgroup>
                <h2>Top Artists</h2>
                <span>from the last 7 days</span>
            </hgroup>
            <ul>
                {#each weeklytopartists as {name, plays, image}, index}
                    <li class="artist">
                        <div class="image" style={`anchor-name: --weekly-artist-anchor-${index}; background-image: url("${image}");`}></div>
                        <p style={`position-anchor: --weekly-artist-anchor-${index}`}>
                            <strong class="artist-name">{name}</strong>
                        </p>
                        <span class="plays" style={`position-anchor: --weekly-artist-anchor-${index}`}>{plays}</span>
                    </li>
                {/each}
            </ul>
            <p>
                <button onclick={shareWeeklyArtists}>Share Top Artists on Twitter</button>
            </p>
        </div>
    </Layout>
</div>

<style>
    .image {
        background-size: cover;
        background-color: #000;
        height: 8rem;
        width: 8rem;
        border-radius: 50%;
    }

    ul {
        list-style-type: none;
        margin: 2rem 0 4rem;
        padding: 0;
        display: flex;
        justify-content: space-between;

        li {
            list-style-type: none;
            margin: 0 .7rem;
        }
    }

    .track, .artist {
        position: relative;

        p {
            position: absolute;
            display: grid;
            top: calc(anchor(bottom) + .5rem);
            left: anchor(left);
            right: anchor(right);
            text-align: center;
            grid-template: 
                "track"
                "name";
        }

        .artist-name {
            font-size: .8rem;
            grid-area: name;
        }

        .track-name {
            font-size: .8rem;
            grid-area: track;
        }

        .plays {
            position: absolute;
            left: calc(anchor(right) - 2rem);
            bottom: calc(anchor(top) - 2rem);
            background-color: #000;
            border-radius: 50%;
            color: #FFF;
            height: 1.7rem;
            width: 1.7rem;
            font-size: .5rem;
            text-align: center;
            display: inline-block;
            line-height: 1.7rem;
            border: 1px solid rgba(188, 188, 188, .5);
        }
    }
</style>
