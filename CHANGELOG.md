# eugène's changelog

## v2

- eugène repo is now `${XDG_CONFIG_HOME:-$HOME/.config}/eugene` by default
- eugène generations dir is now `${XDG_DATA_HOME:-$HOME/.local}/state/eugene` by default
- handlers now match `handlername*` and `$(hostname)_handlername*` files in repo, this enabled host-specific entries
- new `apply` subcommand, equivalent to `eugene build && eugene switch latest`
- `switch` now fails if any handler command fails
- new `align` subcommand, removes gaps in generations numbers, eg. `[0, 2, 3, 6] -> [0, 1, 2, 3]`
- bugfix: correctly raise an error when `toGen` argument is invalid in `diff` subcommand