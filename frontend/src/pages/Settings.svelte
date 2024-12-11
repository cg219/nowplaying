<script lang="ts">
    import Layout from "../lib/Layout.svelte";

    type Props = {
        spotifyOn: boolean
        spotifyUrl: string
        spotifyTrack: boolean
        twitterOn: boolean
        twitterUrl: string
    }

    let { spotifyOn, twitterOn, spotifyUrl, twitterUrl, spotifyTrack }: Props = $props();
    
    const links = [{
        name: "Me",
        url: "/me",
        current: false
    }, {
        name: "Settings",
        url: "/settings",
        current: true
    }]
</script>

<Layout title="Settings" subtitle="Checkin" {links}>
    <form>
        {#if spotifyOn}
            <fieldset>
                <label for="spotify-session">Scrobble Spotify</label>
                <input type="checkbox" name="spotify-session" role="switch" bind:checked={spotifyTrack}>
            </fieldset>
        {:else}    
        <fieldset>
            <label for="spotify-auth">Allow Spotify Access</label>
            <a target="_self" href={spotifyUrl} aria-label="Authorize with Spotify">
                <input type="button" name="spotify-auth" value="Authorize with Spotify" />
            </a>
        </fieldset>
        {/if}

        {#if twitterOn}
        <fieldset>
            <label for="reset-pass">Reset Password</label>
            <input type="text" name="username" placeholder="Username" />
            <input type="button" name="reset-pass" value="Reset Password"/>
        </fieldset>
        {:else}
        <fieldset>
            <label for="x-auth">Allow X Access</label>
            <a target="_self" href={twitterUrl} aria-label="Authorize with X">
                <input type="button" name="x-auth" value="Authorize with X">
            </a>
        </fieldset>
        {/if}
    </form>
</Layout>
