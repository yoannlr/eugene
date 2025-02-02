package main

const configFileName = "eugene.yml"

const defaultConf = `# eugene sample configuration file
handlers:
  - name: apt_pkgs
    sync: sudo apt update
    # in add and remove commands, %s is replaced with the entries handled by the handler
    add: sudo apt install %s
    remove: sudo apt purge --autoremove %s
    upgrade: sudo apt full-upgrade
    # if multiple, add and remove commands are executed once for every entry (eg. apt install vim jq curl)
    # else, one command is executed for each entry (eg. apt install vim, apt install jq, apt install curl)
    multiple: true
    # run anything before and after switching
    # supports your shell's environment variables and eugene's environment variables
    run_before_switch: echo "$(dpkg -l | wc -l) packages on system"
    run_after_switch: echo "now $(dpkg -l | wc -l) packages on system"
  - name: flatpak
    # commands are litteraly run as sh -c "$cmd", you can therefore use && ; || $()...
    setup: sudo apt install flatpak && flatpak remote-add --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo
    add: flatpak install flathub --noninteractive %s
    remove: flatpak uninstall --noninteractive %s; flatpak uninstall --unused --noninteractive
    multiple: false`

const helpText = `eugene, declare and manage your system installations in generations

Main subcommands you may use the most:

eugene build [comment]
  Creates a new generation with the content of each handler's files in the repo.

eugene list
  Lists the generations available on this host.
  The current one is marked with an arrow.

eugene diff <fromGenA> <toGenB> [handler]
  Shows the difference between generations A and B.
  Optionally, only show the difference for the specified handler.

eugene switch <targetGen> [--dry-run]
  Switches to the target generation (ie. installs and removes entries based on diff from the current gen).
  If --dry-run is specified, only shows what would be done.

These are not the only subcommands!
For more information, check out the manual page: man eugene
`

const manURL = "https://raw.githubusercontent.com/yoannlr/eugene/refs/tags/v3/eugene.1"
const manDir = "/usr/local/share/man/man1"

const manText = `
It seems you have not yet installed the manual page.
To install it, please run the following commands:

  sudo mkdir -p ` + manDir + `
  cd ` + manDir + `
  sudo wget ` + manURL + `

`