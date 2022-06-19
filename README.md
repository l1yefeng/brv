# brv
A EPUB reader in web browsers

## Usage

    > brv path/to/book.epub
    book ready at http://localhost:8004
    
Then, go to http://localhost:8004 in your choice of web browser to read the book.

Keyboard controls: 

- Press <kbd>Space</kbd> to show the table of content; press again to show customisation, book info, etc; press again to hide. 
  - Use <kbd>Shift</kbd> to reverse this.
  - Press <kbd>Escape</kbd> to hide and reveal the book.
- Press <kbd>&lt;</kbd> to go to the previous page.
- Press <kbd>&gt;</kbd> to go to the next page.

## Functions of brv

- Remembering last read position.
- Customising reading interface (and remembering it).
- Accessing table of content with one key.

Book page rendering and many functions are built into browsers, such as jumping back and forth in history. 
In addition, some features can be added from the web browser using extensions, 
e.g.

- Dark mode by [Dark Reader](https://darkreader.org)
- Vi keymap by [Vimium](https://github.com/philc/vimium)
- Dictionary by [Dictionary Anywhere](https://github.com/meetDeveloper/Dictionary-Anywhere)

to name a few.
