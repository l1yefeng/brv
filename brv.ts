
let tocOn = false

document.addEventListener("keydown", keyListener)

// space key : toggle toc

const TOC_ID = "brv-toc"

function keyListener(event: KeyboardEvent) {
    switch (event.code) {

        case "Escape":

            event.preventDefault()
            if (tocOn) {
                document.getElementById(TOC_ID).style.display = "none"
                tocOn = false
            }
            break

        case "Space":

            event.preventDefault()

            const elem = document.getElementById(TOC_ID)
            elem.style.display = tocOn ? "none" : "block"
            tocOn = !tocOn

            break
    }
}
