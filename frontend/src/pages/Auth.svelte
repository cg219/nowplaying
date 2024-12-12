<script lang="ts">
    import Layout from "../lib/Layout.svelte";

    let username = $state("")
    let password = $state("")

    async function auth(evt: Event) {
        evt.preventDefault()

        const res = await fetch("/auth/login", {
            method: "POST",
            body: JSON.stringify({
                username,
                password
            })
        })

        const data = await res.json()

        if (data.success) location.pathname = "/me"
    }
</script>

<Layout title="Login" subtitle="Sign into platform">
    <form onsubmit={auth} class="container" id="login" method="POST" action="/api/login">
        <input type="text" name="username" placeholder="Username" bind:value={username} />
        <input type="password" name="password" placeholder="Password" bind:value={password} />
        <button type="submit">Login</button>
    </form>
</Layout>
