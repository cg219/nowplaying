<script>
    import Layout from "../lib/Layout.svelte";

    const links = [{
        name: "Me",
        url: "/me",
        current: true
    }, {
        name: "Settings",
        url: "/settings",
        current: false
    }]
</script>

<Layout title="My Page" subtitle="Checkin" {links}>
    <h1>Last Scrobble</h1>
    <div class="last-scrobble" hx-get="/api/last-scrobble" hx-trigger="load, every 10s" hx-swap="innerHTML">
        <p class="artist"></p>
        <p class="track"></p>
        <p class="date"></p>
    </div>
</Layout>
