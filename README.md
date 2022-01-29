# todo-tracker
Go terminal app powered by [tview](https://github.com/rivo/tview) for managing a limited list of active TODO items.

[![asciicast](https://asciinema.org/a/ZsTp0Gxia4C2IQPIeN3KgNBNw.svg)](https://asciinema.org/a/ZsTp0Gxia4C2IQPIeN3KgNBNw)

## Motivation

Inspired by a suggestion from [4,000 Weeks](https://www.goodreads.com/book/show/54785515-four-thousand-weeks) by Oliver Burkeman, this simple todo list app limits the number of items on a closed list from which all work must be done. The idea is to force intentional prioritization rather than maintaining overly long todo lists and thinking that if only I worked harder or were more efficient, I could get everything done. Life is short (on average around 4,000 weeks ~= 80 years), so we have to choose what is worth our precious time and effort.

## Getting started

To build the app and get started:
`make

tt`

The default location for the backing sqlite db is `~/.todo_tracker.sqlite`, which may be overridden by setting the `TT_DB_FILENAME` environment variable. Note that the app will initialize a db in the given location if it doesn't exist.

The default location for the debug log is `~/.todo_tracker.log`, which may be overridden by setting the `TT_LOG_FILENAME` environment variable.

Other available actions should be apparent - available keyboard shortcuts are visible in the header. For navigating tables and forms, I don't override tview defaults - for forms, that means tab/Shift+tab to move back and forth between form items, enter to select a button, etc; for tables, that means j/k to move up and down, G to jump to the end, and gg to jump to the top.
