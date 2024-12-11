<script lang="ts">
    import Layout from "../lib/Layout.svelte";

    type Props = {
        valid: boolean
        reset: string
        username: string
    }

    let { valid, username, reset }: Props  = $props()
</script>

<Layout title="Reset" subtitle="Checkin">
    <main class="container">
        <header>
            <hgroup>
                <h1>Now Playing</h1>
                <p>Track and Share what you're listening to</p>
            </hgroup>
        </header>
        <section>
            <h1>Reset Password</h1>
            {#if valid}
                <form class="container" id="reset" method="POST" action="/api/reset-password">
                    <input type="hidden" name="username" value={username} />
                    <input type="hidden" name="reset" value={reset}/>
                    <input type="password" name="password" placeholder="Password" />
                    <input type="password" name="password-confirm" placeholder="Confirm Password" />
                    <button type="submit">Reset Password</button>
                </form>
            {:else}
                <p>Invalid Reset Link</p>
                <a href="/">Go Back Home and Login</a>
            {/if}
        </section>
    </main>
</Layout>

