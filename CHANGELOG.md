# eugène's changelog

## v3

- new `storage` subcommand (`storage put` and `storage get`), enables data storage inside generations
- new `repair` subcommand, ensures all entries of each handler for the current generation are installed (in case of change outside of eugene)
- `diff` subcommand now takes an optional `handler` argument
- remove the use of `EUGENE_HOOKS` environment variable
- add man page
- make status codes consistent: 0 is ok, 1 is process error, 2 is command (user) error
- prettier logging
- add `run_if` parameter in handler config: the handler will only trigger if the command is successful

## v2

- eugène repo is now `${XDG_CONFIG_HOME:-$HOME/.config}/eugene` by default
- eugène generations dir is now `${XDG_DATA_HOME:-$HOME/.local}/state/eugene` by default
- handlers now match `handlername*` and `$(hostname)_handlername*` files in repo, this enabled host-specific entries
- new `apply` subcommand, equivalent to `eugene build && eugene switch latest`
- `switch` now fails if any handler command fails
- new `align` subcommand, removes gaps in generations numbers, eg. `[0, 2, 3, 6] -> [0, 1, 2, 3]`
- bugfix: correctly raise an error when `toGen` argument is invalid in `diff` subcommand
- when switching to a new generation, the following environment variables can now be used in scripts:
   - `EUGENE_CURRENT_GEN`, the current generation, eg. `1`
   - `EUGENE_TARGET_GEN`, the target generation, eg. `2`
   - `EUGENE_HANDLER_NAME`, the currently running handler, eg. `apt_pkgs`
- new `deletedups` subcommand, automatically deletes duplicate generations
- new `rollback` subcommand, switches back to `n` generations ago
- automatically generated configuration file is now set to mode 644 (not executable anymore)
