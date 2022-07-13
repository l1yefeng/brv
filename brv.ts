
// these declared variables are inserted in main.go
declare const lastRead: LastRead | null
declare const nextHref: string
declare const prevHref: string

interface LastRead {
    href: string | undefined;
    position: number | undefined;
    "padding-left": string;
    "padding-right": string;
    "font-family": string;
    "font-size": string;
    "line-height": string;
}

// a toc navigation point, for those whose target elem is on current page
interface TocPoint {
    anchor: HTMLAnchorElement;  // <a> in toc, constant
    pos: number;                // y-position of target of <a>
}

// customisation option
interface CustomiseOpt {
    input: HTMLInputElement;            // <input>, constant
    cssKey: string;                     // <body> css property name it controls
    originalValue: string;              // value set in the epub
    setter: (value: string) => string;  // how to compute css value from input
}

const customiseOpts: CustomiseOpt[] = [
    initCustomiseOpt("brv-left-margin", v => v+"%"),
    initCustomiseOpt("brv-right-margin", v => v+"%"),
    initCustomiseOpt("brv-font-family", v => v),
    initCustomiseOpt("brv-font-size", v => v+"%"),
    initCustomiseOpt("brv-line-height", v => v),
]

const boxElem = document.getElementById("brv-box")!
const tocElem = document.getElementById("brv-toc")!
const ciElem = document.getElementById("brv-ci")!

//
// setup
//

// the app box is the toc/customisation/info/about modal
// that floats on top of book pages
//  0: hidden
//  1: toc
//  2: customisation + info + about
let appBoxState = 0

// toc points for this page
// the .pos will change whenever the page resizes or the typography changes
let tocPoints = makeTocPoints()

// use the customisation saved from last read
applyLastRead()

// highlight current point in toc
highlightToc()

// make it respond to keydown
document.addEventListener("keydown", onKeyDown)

// make it respond to scrolling
document.addEventListener("scroll", debounce(onPageShift))

// make it respond to page resize
window.addEventListener("resize", debounce(onPageReformat))

// going to a target on the same page should hide toc
tocPoints.forEach(({anchor}) => {
    anchor.addEventListener("click", hideAppBox)
})

// clicking the background should hide box
boxElem.addEventListener("click", event => {
    if (event.target == boxElem) {
        hideAppBox()
    }
})

setupCustomiseControl()

//
// functions
//

function onKeyDown(event: KeyboardEvent) {
    if (event.target instanceof Element &&
        event.target.tagName.toLowerCase() == "input") {
        return
    }

    switch (event.key) {

        case "Escape":

            event.preventDefault()
            if (appBoxState > 0) {
                hideAppBox()
            }
            break

        case " ":

            event.preventDefault()

            const displayValues = [
                ["none", "", ""],
                ["block", "block", "none"],
                ["block", "none", "block"],
            ]

            // show/hide the box according to appBoxState
            appBoxState += event.shiftKey ? 2 : 1
            appBoxState %= 3;
            [
                boxElem.style.display,
                tocElem.style.display,
                ciElem.style.display,
            ] = displayValues[appBoxState]

            break

        case "<":
            event.preventDefault()
            if (prevHref) {
                window.location.pathname = "/" + prevHref
            } else {
                window.alert("No page before this.")
            }
            break

        case ">":
            event.preventDefault()
            if (nextHref) {
                window.location.pathname = "/" + nextHref
            } else {
                window.alert("No page behind this.")
            }
            break

    }
}

function hideAppBox() {
    boxElem.style.display = "none"
    appBoxState = 0
}

// do what's necessary when the page shifts:
//  update and highlight current toc point, save position for later
// should be called when page is shifted due to scrolling and more
function onPageShift() {

    // re-highlight
    const curr = highlightToc()

    // update location as page shifts
    if (tocPoints.length > 1 && curr) {
        const href = curr.href
        const hashIdx = href.indexOf("#")
        if (hashIdx >= 0) {

            // location
            const id = href.substring(hashIdx+1)
            const elem = document.getElementById(id)
            if (elem != null) {
                elem.removeAttribute("id")
                window.location.hash = "#" + id
                elem.id = id
            }

            // title
            document.title = curr.textContent
        }
    } else if (tocPoints.length == 1) {
        document.title = tocPoints[0].anchor.textContent
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

// return the <a> point which is considered currently being read
function currentTocPoint(): HTMLAnchorElement {

    // section is being read if its top y-position is past mid
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

// highlight (by updating elem class) current point.
// return it if exists.
function highlightToc() : HTMLAnchorElement | null {
    if (tocPoints.length == 0) {
        return null
    }

    const a = tocPoints.length == 1 ? tocPoints[0].anchor : currentTocPoint()

    const className = "current"
    tocPoints.forEach(({anchor}) => {
        anchor.parentElement!.classList.remove(className)
    })
    a.parentElement!.classList.add(className)

    return a
}

// set .anchor (always the same on every call).
// set .pos, calculate the position of each toc target on the current page.
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
        if (input.type == "number" && inValue != "") {
            const num: number = +inValue
            inValue = num.toString()
        }
        input.value = inValue
        // apply
        document.body.style[cssKey] = inValue ? setter(inValue) : originalValue
    })

    onPageReformat()
}

function applyLastRead() {

    // scroll to last read
    if (lastRead) {
        customiseOpts.forEach(({input, cssKey}) => {
            input.value = lastRead[cssKey]
        })
    }
    // otherwise, default values in config.html are used

    // apply to page
    applyConfig()

    if (lastRead && lastRead.position) {
        window.scrollTo(0, scrollPositionFromLastRead(lastRead.position))
    }
}

function setupCustomiseControl() {
    const applyBtn = document.getElementById("brv-apply-config")!
    const okBtn = document.getElementById("brv-ok-config")!

    // setup inputs respond to enter
    customiseOpts.forEach(({input}) => {
        input.addEventListener("keydown", event => {
            if (event.key == "Enter") {
                event.preventDefault();
                (event.shiftKey ? applyBtn : okBtn).click()
            }
        })
    })

    // right margin changes with left if they are the same
    const mlElem = customiseOpts[0].input
    const mrElem = customiseOpts[1].input
    const updateMaster = () => {
        mlElem.dataset.master = mlElem.value == mrElem.value ? "1" : "0"
    }
    updateMaster()
    mlElem.addEventListener("input", () => {
        if (mlElem.dataset.master == "1") {
            mrElem.value = mlElem.value
        }
    });
    [mlElem, mrElem].forEach(elem => {
        elem.addEventListener("change", updateMaster)
    })

    // setup buttons respond to click
    applyBtn.addEventListener("click", function() {
        applyConfig()
        saveLastRead()
    })
    okBtn.addEventListener("click", function() {
        applyConfig()
        hideAppBox()
        saveLastRead()
    })
}

function initCustomiseOpt(id: string, setter: (value: string) => string): CustomiseOpt {
    const input = document.getElementById(id) as HTMLInputElement
    const cssKey = input.name
    const originalValue = document.body.style[cssKey]
    return {input, cssKey, originalValue, setter}
}

function saveLastRead() {

    let lastRead = {
        href: window.location.pathname,
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
    const pos = window.scrollY + window.innerHeight/4
    return pos / document.body.clientHeight
}

// the inverse of readingPosition()
function scrollPositionFromLastRead(lastReadPosition: number): number {
    const pos = document.body.clientHeight * lastReadPosition
    return pos - window.innerHeight/4
}

// return a version of fn that won't be called too often
function debounce(fn: () => void, wait: number = 200): () => void {
    let timeout: number;

    return function() {
        const later = () => {
            clearTimeout(timeout)
            fn()
        }

        clearTimeout(timeout)
        timeout = setTimeout(later, wait)
    }
}
