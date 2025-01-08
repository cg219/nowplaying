<script lang="ts">
    import Layout from "../lib/Layout.svelte";
    import type { Link } from "../lib/customtypes";

    type Props = {
        spotifyOn: boolean
        spotifyUrl: string
        spotifyTrack: boolean
        twitterOn: boolean
        twitterUrl: string
        links: Link[]
        title: string
        subtitle: string
    }

    let apikey = $state("")
    let apiname = $state("")

    async function getData() {
        console.log("dataaa")
        const res = await fetch("/api/settings", {
            method: "POST",
            credentials: "same-origin"
        })

        const data = await res.json()

        return data as Props
    }

    async function toggleScrobble(evt) {
        if (evt.target.checked) {
            await fetch("/api/spotify", {
                method: "POST",
                credentials: "same-origin"
            })
        } else {
            await fetch("/api/spotify", {
                method: "DELETE",
                credentials: "same-origin"
            })
        }
    }

    async function resetPassword() {
        const data = new URLSearchParams();
        const username = document.querySelector('input[name="username"]').value;

        data.append("username", username)

        await fetch("/api/forgot-password", {
            headers: {
                "Content-type": "application/x-www-form-urlencoded"
            },
            method: "POST",
            body: data
        })
    }

    async function generateKey() {
        const res = await fetch(`/api/generate-apikey/${apiname}`, { method: "POST" }).then((res) => res.json())

        apikey = res.apikey
    }
</script>

{#await getData()}
    <h1>loading...</h1>
{:then data} 
    <Layout title={data.title} subtitle={data.subtitle} links={data.links}>
        <form>
            {#if data.spotifyOn}
                <fieldset>
                    <label for="spotify-session">Scrobble Spotify</label>
                    <input type="checkbox" onchange={toggleScrobble}  name="spotify-session" role="switch" bind:checked={data.spotifyTrack}>
                </fieldset>
            {:else}    
                <fieldset>
                    <label for="spotify-auth">Allow Spotify Access</label>
                    <a target="_self" href={data.spotifyUrl} aria-label="Authorize with Spotify">
                        <input type="button" name="spotify-auth" value="Authorize with Spotify" />
                    </a>
                </fieldset>
            {/if}

            {#if data.twitterOn}
                <fieldset>
                    <label for="reset-pass">Reset Password</label>
                    <input type="text" name="username" placeholder="Username" />
                    <input type="button" onclick={resetPassword} name="reset-pass" value="Reset Password"/>
                </fieldset>
            {:else}
                <fieldset>
                    <label for="x-auth">Allow X Access</label>
                    <a target="_self" href={data.twitterUrl} aria-label="Authorize with X">
                        <input type="button" name="x-auth" value="Authorize with X">
                    </a>
                </fieldset>
            {/if}
            <fieldset>
                <label for="new-key">New API Key</label>
                <input type="text" placeholder="Name" bind:value={apiname}>
                <input type="button" onclick={generateKey} name="api-generate" value="Generate">
                <input type="text" name="new-key" disabled value={apikey}>
            </fieldset>
        </form>
    </Layout>
{/await}
