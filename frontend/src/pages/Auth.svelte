<script lang="ts">
    import Layout from "../lib/Layout.svelte";
</script>

<Layout title="Login" subtitle="Checkin">
    <form class="container" id="login" method="POST" action="/api/login">
        <input type="text" name="username" placeholder="Username" />
        <input type="password" name="password" placeholder="Password" />
        <button type="submit">Login</button>
    </form>
</Layout>
