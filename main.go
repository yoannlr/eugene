package main

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "slices"
    "strings"
    "os/exec"

    "gopkg.in/yaml.v2"
)

const textReset = "\033[0m"
const textCyan = "\033[1;36m"
const textGreen = "\033[0;32m"
const textRed = "\033[0;31m"
const textYellow = "\033[0;33m"
const textPurple = "\033[1;35m"
const textBold = "\033[1m"

// utils

func fileExists(f string) bool {
    _, err := os.Stat(f)
    return err == nil
}

func commandExec(shellCommand string) bool {
    cmd := exec.Command("sh", "-c", shellCommand)
    cmd.Env = os.Environ()
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    err := cmd.Run()
    return err == nil
}

func hasDryRunFlag(args []string, startLookup int) bool {
    for i := startLookup; i < len(args); i++ {
        if args[i] == "--dry-run" {
            return true
        }
    }
    return false
}

func configInit(repo string) {
    defaultConf := `# eugene sample configuration file
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
    outFile := filepath.Join(repo, "eugene.yml")
    os.WriteFile(outFile, []byte(defaultConf), os.ModePerm)
    eugeneMessage("Created default config file to " + outFile)
}

// logging

func eugeneMessage(msg string) {
    fmt.Println(textGreen + "Info  | " + textReset + msg)
}

func eugeneError(msg string) {
    fmt.Println(textRed + "Error | " + msg + textReset)
}

func eugeneDebug(msg string) {
    fmt.Println("Debug | " + msg)
}

func eugeneUsage(showType bool, msg string) {
    if showType {
        fmt.Println(textRed + "Usage | " + textReset + msg)
    } else {
        fmt.Println(textRed + "      | " + textReset + msg)
    }
}

// main

type Handler struct {
    Name string `yaml:"name"`
    Add string `yaml:"add"`
    Remove string `yaml:"remove"`
    Sync string `yaml:"sync"`
    Upgrade string `yaml:"upgrade"`
    Multiple bool `yaml:"multiple"`
    Setup string `yaml:"setup"`
    HookPre string `yaml:"run_before_switch"`
    HookPost string `yaml:"run_after_switch"`
}

type Config struct {
    // avec une map[string]Handler, l'ordre n'est pas respecte
    Handlers []Handler `yaml:"handlers"`
}

func main() {
    repo := os.Getenv("EUGENE_REPO")
    if repo == "" {
        dotConfig := os.Getenv("XDG_CONFIG_HOME")
        if dotConfig == "" {
            home, _ := os.UserHomeDir()
            dotConfig = filepath.Join(home, ".config")
        }
        repo = filepath.Join(dotConfig, "eugene")
        os.Setenv("EUGENE_REPO", repo)
    }
    if ! fileExists(repo) {
        os.Mkdir(repo, os.ModePerm)
        eugeneMessage("Initialized new repository to " + repo)
        configInit(repo)
    }

    gens := os.Getenv("EUGENE_GENS")
    if gens == "" {
        dotLocal := os.Getenv("XGD_DATA_HOME")
        if dotLocal == "" {
            home, _ := os.UserHomeDir()
            dotLocal = filepath.Join(home, ".local")
        }
        dotLocalState := filepath.Join(dotLocal, "state")
        gens = filepath.Join(dotLocalState, "eugene")
        // gens = filepath.Join(repo, ".gens")
        os.Setenv("EUGENE_GENS", gens)
    }

    if ! fileExists(gens) {
        os.Mkdir(gens, os.ModePerm)
        genCreate(gens, 0, "Empty generation (automatically created)")
        genSetCurrent(gens, 0)
        genSetLatest(gens, 0)
        eugeneMessage("Initialized generations directory to " + gens)
    }

    hooks := os.Getenv("EUGENE_HOOKS")
    if hooks == "" {
        hooks = filepath.Join(repo, "hooks")
        os.Setenv("EUGENE_HOOKS", hooks)
    }

    if len(os.Args) < 2 {
        eugeneUsage(true, "eugene <build|list|diff|switch|delete|show|apply|align>")
        eugeneUsage(false, "")
        eugeneUsage(false, textBold + "eugene build [comment]" + textReset)
        eugeneUsage(false, "  creates a new generation with the current file")
        eugeneUsage(false, "  the generation is automatically removed if it does not differ from the latest")
        eugeneUsage(false, textBold + "eugene list" + textReset)
        eugeneUsage(false, "  lists all the generations")
        eugeneUsage(false, "  the current one is indicated with an arrow")
        eugeneUsage(false, textBold + "eugene diff <fromGen> <toGen>" + textReset)
        eugeneUsage(false, "  shows the difference between two generations")
        eugeneUsage(false, "  exit code is 0 if they're the same, 1 if they're different")
        eugeneUsage(false, textBold + "eugene switch <targetGen> [--dry-run]" + textReset)
        eugeneUsage(false, "  switches to the target generation (sync/remove/install packages)")
        eugeneUsage(false, textBold + "eugene delete <gen1> [gen2 gen3 ...]" + textReset)
        eugeneUsage(false, "  deletes one or more generations")
        eugeneUsage(false, "  deletion of the current generation and generation 0 is forbidden")
        eugeneUsage(false, textBold + "eugene show <gen> [handler]" + textReset)
        eugeneUsage(false, "  show all the entries in a generation")
        eugeneUsage(false, textBold + "eugene upgrade [--dry-run]" + textReset)
        eugeneUsage(false, "  perform upgrade command of each handler")
        eugeneUsage(false, textBold + "eugene apply [--dry-run]" + textReset)
        eugeneUsage(false, "  build a new generation and automatically switch to it")
        eugeneUsage(false, "  equivalent to `eugene build && eugene switch latest`")
        eugeneUsage(false, textBold + "eugene align" + textReset)
        eugeneUsage(false, "  remove gaps in generations numbers, eg. [0, 2, 3, 6] -> [0, 1, 2, 3]")
        eugeneUsage(false, "")
        eugeneUsage(false, "eugene can be configured with the following environment variables")
        eugeneUsage(false, "  - EUGENE_REPO - list of entries for each handler, defaults to ~/.config/.eugene")
        eugeneUsage(false, "  - EUGENE_GENS - internal storage for generations, defaults to ${EUGENE_REPO}/.gens")
        eugeneUsage(false, "  - EUGENE_HOOKS - hook scripts for each handler, defaults to ${EUGENE_REPO}/hooks")
        eugeneUsage(false, "if not user-defined, these variables are automatically set by eugene at runtime")
        eugeneUsage(false, "and can be used in your custom scripts/hooks")
        os.Exit(1)
    }

    configFile := filepath.Join(repo, "eugene.yml")
    if ! fileExists(configFile) {
        eugeneError("Configuration file " + configFile + " not found")
        os.Exit(1)
    }

    data, _ := os.ReadFile(configFile)
    var config Config
    yaml.Unmarshal(data, &config)

    if os.Args[1] == "list" {
        allGens := genGetAll(gens)
        currentGen := genGetCurrent(gens)
        for _, num := range allGens {
            if num == currentGen {
                fmt.Print(textGreen + "->")
            } else {
                fmt.Print("  ")
            }

            fmt.Print(" " + strconv.Itoa(num) + " ")

            comment := genGetComment(gens, num)

            if comment != "" {
                fmt.Print(string(comment))
            } else {
                fmt.Print("(no comment)")
            }

            fmt.Print(textReset + "\n")
        }
    } else if os.Args[1] == "build" {
        if doBuild(os.Args, repo, gens, config) {
            os.Exit(0)
        } else {
            os.Exit(1)
        }
    } else if os.Args[1] == "delete" {
        if len(os.Args) > 1 {
            deleteGens := os.Args[2:]
            // les mettre dans l'ordre permet de ne faire qu'un seul changement du pointeur latest (si besoin)
            slices.Sort(deleteGens)
            for _, g := range deleteGens {
                num := genParse(gens, g)
                if ! genDelete(gens, num) {
                    eugeneError("Error deleting generation " + g)
                    os.Exit(1)
                }
            }
        }
    } else if os.Args[1] == "diff" {
        if len(os.Args) != 4 {
            eugeneUsage(true, "eugene diff <genA> <genB>")
            os.Exit(1)
        }
        
        genA := genParse(gens, os.Args[2])
        genB := genParse(gens, os.Args[3])
        if genA == -1 {
            eugeneError("Generation " + os.Args[2] + " is invalid or does not exist")
            os.Exit(2)
        }
        if genB == -1 {
            eugeneError("Generation " + os.Args[3] + " is invalid or does not exist")
            os.Exit(2)
        }

        hasDiff := false
        for _, h := range config.Handlers {
            handlerStatus(h, "diff")
            add, remove := genDiff(gens, genA, genB, h)
            if ! hasDiff && (len(add) > 0 || len(remove) > 0) {
                hasDiff = true
            }
            if h.Multiple {
                fmt.Println(textRed + "- " + strings.Join(remove, " ") + textReset)
                fmt.Println(textGreen + "+ " + strings.Join(add, " ") + textReset)
            } else {
                for _, entry := range remove {
                    fmt.Println(textRed + "- " + entry + textReset)
                }
                for _, entry := range add {
                    fmt.Println(textGreen + "+ " + entry + textReset)
                }
            }
        }

        if hasDiff {
            eugeneMessage("Generations differ")
            os.Exit(1)
        } else {
            eugeneMessage("Generations are identical")
            os.Exit(0)
        }
    } else if os.Args[1] == "switch" {
        if len(os.Args) < 3 {
            eugeneUsage(true, "eugene switch <targetGen> [--dry-run]")
            os.Exit(1)
        }

        targetGen := genParse(gens, os.Args[2])
        if targetGen == -1 {
            eugeneError("The target generation is invalid or does not exist")
            os.Exit(1)
        }
        if targetGen == genGetCurrent(gens) {
            eugeneError("Switching to the current generation makes no sense")
            os.Exit(1)
        }

        dryRun := hasDryRunFlag(os.Args, 3)

        if doSwitch(config, gens, targetGen, dryRun) {
            os.Exit(0)
        } else {
            os.Exit(1)
        }
    } else if os.Args[1] == "show" {
        if len(os.Args) < 3 {
            eugeneUsage(true, "eugene show <gen> [handler]")
            os.Exit(1)
        }
        num := genParse(gens, os.Args[2])
        if num == -1 {
            eugeneError("Generation '" + os.Args[2] + "' is invalid or does not exist")
            os.Exit(1)
        }
        handler := ""
        if len(os.Args) == 4 {
            handler = os.Args[3]
        }

        for _, h := range config.Handlers {
            if handler != "" && h.Name != handler {
                continue
            }
            handlerStatus(h, "show")
            entries := handlerGetEntries(gens, num, h)
            if len(entries) > 0 {
                if h.Multiple {
                    fmt.Println("* " + strings.Join(entries, " "))
                } else {
                    for _, e := range entries {
                        fmt.Println("* " + e)
                    }
                }
            }
        }
    } else if os.Args[1] == "upgrade" {
        dryRun := hasDryRunFlag(os.Args, 2)
        eugeneMessage("Running upgrade")
        for _, h := range config.Handlers {
            handlerStatus(h, "upgrade")
            if h.Upgrade == "" {
                handlerMessage(h, "Command undefined")
            } else {
                handlerExec(h.Upgrade, dryRun)
            }
        }
    } else if os.Args[1] == "apply" {
        dryRun := hasDryRunFlag(os.Args, 2)
        if doBuild(make([]string, 0), repo, gens, config) {
            latestGen := genGetLatest(gens)
            doSwitch(config, gens, latestGen, dryRun)
        } else {
            eugeneMessage("Therefore not switching")
        }
    } else if os.Args[1] == "align" {

    }
}
