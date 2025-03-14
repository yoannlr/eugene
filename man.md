EUGENE 1 "JANUARY 2025" Linux "User Manuals"
===

# NAME

eugene - version your system packages (and more!) in generations

# SYNOPSIS

**eugene**
<**subcommand**>

# DESCRIPTION

eugene is a program that enables you to declare and version your system installations in generations.
You can switch back and forth between generations, the changes will automatically be applied.

The source code can be found in the repository at https://github.com/yoannlr/eugene

The installations are managed with handlers.
Each handler is responsible for a specific kind of installation you define: apt packages, flatpaks, gsettings values...

Everything you declare in eugene is stored in a eugene repository, usually located at `~/.config/eugene`.

# CONFIGURATION

Handlers are declared in eugene's configuration file, usually `~/.config/eugene/eugene.yml`.
eugene will generate a sample config file for you on it's first execution.

In the `handlers` section of the configuration file, each handler is made out of the following fields:

```
handlers:
  - name: handler_name
    run_if: command that determines if the handler will run
    setup:
      - when: command to detect a specific environment
        run: setup command for that environment
      - when: command to detect a specific environment
        run: setup command for that environment
    sync: handler sync command
    add: handler add command
    remove: handler remove command
    multiple: true/false
    run_before_switch: hook command
    run_after_switch: hook command
```

Here's an example for a `apt_pkgs` handler:

```
handlers:
  - name: apt_pkgs
    run_if: which apt > /dev/null
    sync: sudo apt update
    add: sudo apt install %s
    remove: sudo apt purge --autoremove %s
    upgrade: sudo apt full-upgrade
    multiple: true
    run_before_switch: echo "$(dpkg -l | wc -l) packages on system"
    run_after_switch: echo "now $(dpkg -l | wc -l) packages on system"
```

All the commands are executed as `sh -c "command"`.

`%s` in add and remove commands will be replaced with handler entries.

If multiple is set to true, `%s` will be replaced with all the entries separated with a space and only one command will be run.
Otherwise, one command will be run for each entry.

Every handler will match the files beginning with it's name in the eugene repository.
You can also prefix the handler's name with a hostname in the repo, the handler will match these files only on the correct host.

Example with `apt_pkgs` handler:

- `apt_pkgs`: matches everywhere
- `flatpak`: does not match
- `apt_pkgs_libvirt`: matches everywhere
- `x220_apt_pkgs_i3`: only matches on `x220` host

# OPTIONS

The following subcommands are available:

`eugene build [comment]`
  Builds a new generation with the entries of each handler.
  You can optionnally add a description to the generation with a comment.
  If the newly built generation does not differ from the latest, it is automatically removed.

`eugene list [--with-hash]`
  Lists all the generations.
  The current one is indicated with an arrow.
  If `--with-hash` specified, shows the generation's hash.

`eugene diff <fromGenA> <toGenB> [handler]`
  Shows the difference between two generations (what would be done if you switch from gen A to gen B).
  If handler is specified, only shows the diff for this handler.

`eugene switch <toGen> [--dry-run]`
  Switches to a new generation, ie. performs remove and add commands for each handler according to the diff between the target generation and the current generation.
  If `--dry-run` specified, only show what would be done.

`eugene delete <genA> [genB genC ...]`
  Deletes one or more generations.
  For consistency reasons, generation 0 and the current generation can not be deleted.

`eugene show <gen> [handler]`
  Show the entries managed by each handler in the target generation.
  If handler is specified, only shows the entries for this handler.

`eugene upgrade [--dry-run]`
  Runs each handler upgrade command.

`eugene apply [--dry-run]`
  Equivalent to `eugene build && eugene switch latest`.

`eugene align [--dry-run]`
  Removes gaps in generations numbers, eg `eg. [0, 2, 3, 6] -> [0, 1, 2, 3]`.

`eugene deletedups [--dry-run] [--align]`
  Delete duplicates generations based on hashes.
  If `--align` specified, aligns the generations after deleting duplicates.

`eugene rollback [n [--dry-run]]`
  Rolls back (ie. switches to) n generations ago.
  If n is not specified, rolls back to the previous generation.

`eugene repair [--dry-run]`
  Ensures every handler entry is satisfied.
  Equivalent to switching from generation 0 to the current one, ie. running every handler add command for every entry of the current generation.
  Useful if an entry was changed outside of eugene.

`eugene storage put <gen> <namespace> <key> [value]`
  Stores data in the target generation.
  If value is not specified, eugene will attempt to read from standard input.

`eugene storage get <gen> <namespace> <key>`
  Retreives data stored in the target generation.
  If namespace/key does not match any data, returns nothing but exit code remains **0**.

# EXIT STATUS

A value of **0** is returned if everything went well.

If something went wrong when performing a command, a value of **1** is returned.

If the command is incorrect (user error), a value of **2** is returned.

Exception: the diff subcommand returns **0** if the generations are identical, **1** if they differ.

# ENVIRONMENT

eugene can be configured with the following environment variables:

`EUGENE_REPO`
  Path to configuration file and the entry files of each handler.
  Defaults to `${XDG_CONFIG_HOME-$HOME/.config}/eugene`.

`EUGENE_GENS`
  Internal storage for generations.
  Defaults to `${XDG_DATA_HOME-$HOME/.local}/state/eugene`
  **DO NOT EDIT** the files in this directory.

When performing a switch operation, eugene exports the following environment variables for use in handler commands/scrips:

`EUGENE_CURRENT_GEN`
  The current generation, eg. `1`.

`EUGENE_TARGET_GEN`
  The target generation, eg. `2`.

`EUGENE_HANDLER_NAME`
  The name of the currently running handler, eg. `apt_pkgs`.

# AUTHORS

yoannlr (https://github.com/yoannlr)
