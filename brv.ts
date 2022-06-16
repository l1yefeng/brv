
interface TocPoint {
    anchor: HTMLAnchorElement;
    pos: number;
}

const boxElem = document.getElementById("brv-box")
const tocElem = document.getElementById("brv-toc")
const ciElem = document.getElementById("brv-ci")

// setup

let appBoxState = 0
let tocPoints = makeTocPoints()

// set click handlers of some toc anchors
tocPoints.forEach(({anchor}) => {
    anchor.addEventListener("click", () => {
        // hide toc
        boxElem.style.display = "none"
        appBoxState = 0
    })
})

// respond to keys
document.body.addEventListener("keydown", handleKeyDown)

// initial highlight current point in toc
// respond to scrolling
if (tocPoints.length > 0) {
    if (tocPoints.length == 1) {
        highlightToc(tocPoints[0].anchor)
    } else {
        const selected = findCurrentTocPoint()
        highlightToc(selected)
        document.addEventListener("scroll", throttle(handleScroll, 150))
    }
}

// respond to resize
window.addEventListener("resize", throttle(handleResize, 150))

// functions

function handleKeyDown(event: KeyboardEvent) {
    if (event.target instanceof Element &&
        event.target.tagName.toLowerCase() == "input") {
        return
    }

    switch (event.code) {

        case "Escape":

            event.preventDefault()
            if (appBoxState > 0) {
                boxElem.style.display = "none"
                appBoxState = 0
            }
            break

        case "Space":

            event.preventDefault()

            const displayValues = [
                ["none", undefined, undefined],
                ["block", "block", "none"],
                ["block", "none", "block"],
            ]

            appBoxState += event.shiftKey ? 2 : 1
            appBoxState %= 3;
            [boxElem.style.display, tocElem.style.display, ciElem.style.display] = displayValues[appBoxState]

            break
    }
}

function handleScroll() {

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

function handleResize() {

    // re-calculate toc target positions
    tocPoints = makeTocPoints()

    // re-highlight
    handleScroll()
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
