
let tocOn = false

document.addEventListener("keydown", keyListener)

// space key : toggle toc

const KEYCODE_SPACE = 'Space'

const TOC_ID = "brv-toc"

function keyListener(event: KeyboardEvent) {
    if (event.code != KEYCODE_SPACE) {
        return
    }

    event.preventDefault()

    const elem = document.getElementById(TOC_ID)
    elem.style.display = tocOn ? "none" : "block"
    tocOn = !tocOn
}

function makeToc(): HTMLDivElement {
    const elem = document.createElement("div")

    elem.id = TOC_ID

    elem.style.display = "block"
    elem.style.position = "fixed"
    elem.style.height = "80%"
    elem.style.width = "50%"
    elem.style.left = "25%"
    elem.style.top = "10%"
    elem.style.background = "rgba(255, 255, 255, .5)"

    return elem
}