htmx.onLoad(() => {
    htmx.on('input[name="spotify-session"]', "change", async (evt) => {
        if (evt.target.checked) {
            await fetch("/api/spotify", {
                method: "POST"
            })
        } else {
            await fetch("/api/spotify", {
                method: "DELETE"
            })
        }
        console.log(evt.target.checked)
    })
    htmx.on('input[name="reset-pass"]', "click", async(evt) => {
        const data = new URLSearchParams();
        const username = htmx.find('input[name="username"]').value;

        data.append("username", username)

        await fetch("/api/forgot-password", {
            headers: {
                "Content-type": "application/x-www-form-urlencoded"
            },
            method: "POST",
            body: data
        })
    })
})

