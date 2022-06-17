
declare const lastRead: {}

interface TocPoint {
    anchor: HTMLAnchorElement;
    pos: number;
}

interface CustomiseOpt {
    input: HTMLInputElement;
    cssKey: string;
    originalValue: string;
    setter: (value: string) => string;
}

const boxElem = document.getElementById("brv-box")!
const tocElem = document.getElementById("brv-toc")!
const ciElem = document.getElementById("brv-ci")!

const customiseOpts: CustomiseOpt[] = [
    initCustomiseOpt("brv-left-margin", v => v+"%"),
    initCustomiseOpt("brv-right-margin", v => v+"%"),
    initCustomiseOpt("brv-font-family", v => v),
    initCustomiseOpt("brv-font-size", v => v+"%"),
    initCustomiseOpt("brv-line-height", v => v),
]

//
// setup
//

let appBoxState = 0
let tocPoints = makeTocPoints()

// set click handlers of some toc anchors
tocPoints.forEach(({anchor}) => {
    anchor.addEventListener("click", () => {
        hideAppBox()
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
        highlightToc(findCurrentTocPoint())
        document.addEventListener("scroll", throttle(onPageShift, 150))
    }
}

// respond to resize
window.addEventListener("resize", throttle(onPageReformat, 150))

// config setup
config()

//
// functions
//

function handleKeyDown(event: KeyboardEvent) {
    if (event.target instanceof Element &&
        event.target.tagName.toLowerCase() == "input") {
        return
    }

    switch (event.code) {

        case "Escape":

            event.preventDefault()
            if (appBoxState > 0) {
                hideAppBox()
            }
            break

        case "Space":

            event.preventDefault()

            const displayValues = [
                ["none", "", ""],
                ["block", "block", "none"],
                ["block", "none", "block"],
            ]

            appBoxState += event.shiftKey ? 2 : 1
            appBoxState %= 3;
            [boxElem.style.display, tocElem.style.display, ciElem.style.display] = displayValues[appBoxState]

            break
    }
}

function hideAppBox() {
    boxElem.style.display = "none"
    appBoxState = 0
}

function onPageShift() {

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

    // save last read
    saveLastRead()

}

function onPageReformat() {

    // re-calculate toc target positions
    tocPoints = makeTocPoints()

    // re-highlight
    onPageShift()
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
        anchor.parentElement!.classList.remove(className)
    })
    a.parentElement!.classList.add(className)
}

// calculate the position of each toc target on the current page
function makeTocPoints(): TocPoint[] {
    const pageHref = window.location.pathname
    const anchors = tocElem.querySelectorAll<HTMLAnchorElement>(`a[href^="${pageHref}"]`)

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

function applyConfig() {
    customiseOpts.forEach(({input, cssKey, originalValue, setter}) => {
        // clean
        let inValue = input.value.trim()
        if (input.type == "number") {
            const num: number = +inValue
            inValue = num.toString()
        }
        input.value = inValue
        // apply
        document.body.style[cssKey] = inValue ? setter(inValue) : originalValue
    })

    onPageReformat()
}

function config() {

    // use last read if any
    if (lastRead) {
        customiseOpts.forEach(({input, cssKey}) => {
            input.value = lastRead[cssKey]
        })
    }

    // apply to page
    applyConfig()

    // setup buttons respond to click
    document.getElementById("brv-apply-config")!.addEventListener("click", function() {
        applyConfig()
        saveLastRead()
    })
    document.getElementById("brv-ok-config")!.addEventListener("click", function() {
        applyConfig()
        hideAppBox()
        saveLastRead()
    })
}

function initCustomiseOpt(id: string, setter: (value: string) => string): CustomiseOpt {
    const input = document.getElementById(id) as HTMLInputElement
    const cssKey = input.name
    const originalValue = document.body.style[cssKey]
    return { input, cssKey, originalValue, setter }
}

function saveLastRead() {

    let lastRead = {
        position: readingPosition(),
    }
    customiseOpts.forEach(({input, cssKey}) => {
        lastRead[cssKey] = input.value
    })

    // send to server
    fetch("/save_brv", {
        method: "POST",
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(lastRead),
    })
}

function readingPosition(): number {
    // TODO
    return 0
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
