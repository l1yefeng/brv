
interface TocPoint {
    anchor: HTMLAnchorElement;
    pos: number;
}

const boxElem = document.getElementById("brv-box")
const tocElem = document.getElementById("brv-toc")
const ciElem = document.getElementById("brv-ci")

enum AppBoxState {
    None,
    Toc,
    ConfigInfo,
}

// setup

let appBoxState = AppBoxState.None
let tocPoints = makeTocPoints()

// set click handlers of some toc anchors
tocPoints.forEach(({anchor}) => {
    anchor.addEventListener("click", () => {
        // hide toc
        boxElem.style.display = "none"
        appBoxState = AppBoxState.None
    })
})

// respond to keys
document.body.addEventListener("keydown", keyListener)

// initial highlight current point in toc
// respond to scrolling
if (tocPoints.length > 0) {
    if (tocPoints.length == 1) {
        highlightToc(tocPoints[0].anchor)
    } else {
        const selected = findCurrentTocPoint()
        highlightToc(selected)
        document.addEventListener("scroll", throttle(scrollListener, 100))
    }
}

// functions

function keyListener(event: KeyboardEvent) {
    if (event.target instanceof Element && event.target.tagName.toLowerCase() == "input") {
        return
    }

    switch (event.code) {

        case "Escape":

            event.preventDefault()
            if (appBoxState != AppBoxState.None) {
                boxElem.style.display = "none"
                appBoxState = AppBoxState.None
            }
            break

        case "Space":

            event.preventDefault()

            if (appBoxState == AppBoxState.None) {
                boxElem.style.display = "block"
                tocElem.style.display = "block"
                ciElem.style.display = "none"
                appBoxState = AppBoxState.Toc
            } else if (appBoxState == AppBoxState.Toc) {
                tocElem.style.display = "none"
                ciElem.style.display = "block"
                appBoxState = AppBoxState.ConfigInfo
            } else {
                boxElem.style.display = "none"
                appBoxState = AppBoxState.None
            }

            break
    }
}

function scrollListener() {

    // highlight
    const selected = findCurrentTocPoint()
    highlightToc(selected)

    // location
    const href = selected.href
    const hashIdx = href.indexOf("#")
    if (hashIdx >= 0) {
        const id = href.substring(hashIdx+1)
        const elem = document.getElementById(id)
        if (elem != null) {
            elem.removeAttribute("id")
            window.location.hash = "#" + id
            elem.id = id
        }
    }

}

function findCurrentTocPoint(): HTMLAnchorElement {

    const mid = window.scrollY + window.innerHeight/2

    let curr: number
    for (curr = 0; curr < tocPoints.length; curr++) {
        const tPos = tocPoints[curr].pos
        if (tPos > mid) {
            break
        }
    }
    curr--

    return tocPoints[curr < 0 ? 0 : curr].anchor
}

function highlightToc(a: HTMLAnchorElement) {
    const className = "current"
    tocPoints.forEach(({anchor}) => {
        anchor.parentElement.classList.remove(className)
    })
    a.parentElement.classList.add(className)
}

// calculate the position of each toc target on the current page
function makeTocPoints(): TocPoint[] {
    const pageHref = window.location.pathname
    const anchors = tocElem.querySelectorAll(`a[href^="${pageHref}"]`)

    let points: TocPoint[] = []
    anchors.forEach((elem: HTMLAnchorElement) => {
        let pos: number

        const href = elem.href
        const hashIdx = href.indexOf("#")
        if (hashIdx < 0) { // base
            pos = 0
        } else {
            const id = href.substring(hashIdx + 1)
            const target = document.getElementById(id)
            pos = target == null ? 0 : target.offsetTop
        }

        points.push({anchor: elem, pos})
    })

    return points
}

function throttle(fn: () => void, wait: number): () => void {
    let waiting = false

    return function() {
        if (waiting) {
            return
        }

        waiting = true
        setTimeout(() => {
            fn.apply(this)
            waiting = false
        }, wait)
    }
}
