package main

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "slices"
    "strings"
    "os/exec"
    "bufio"

    "gopkg.in/yaml.v2"
)

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

func hasFlag(args []string, flag string, startLookup int) bool {
    for i := startLookup; i < len(args); i++ {
        if args[i] == flag {
            return true
        }
    }
    return false
}

func configInit(repo string) {
    outFile := filepath.Join(repo, configFileName)
    os.WriteFile(outFile, []byte(defaultConf), 0644)
    logInfo("Created default config file to " + outFile)
}

func manPageInstalled() bool {
    return fileExists(filepath.Join(manDir, "eugene.1"))
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
        logInfo("Initialized new repository to " + repo)
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
        logInfo("Initialized generations directory to " + gens)
    }

    if len(os.Args) < 2 {
        fmt.Print(helpText)
        if ! manPageInstalled() {
            fmt.Print(manText)
        }
        os.Exit(2)
    }

    configFile := filepath.Join(repo, configFileName)
    if ! fileExists(configFile) {
        logError("Configuration file " + configFile + " not found")
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
                    logError("Error deleting generation " + g)
                    os.Exit(1)
                }
            }
        }
    } else if os.Args[1] == "diff" {
        if len(os.Args) < 4 {
            logUsage("eugene diff <genA> <genB> [handler]")
        }
        
        genA := genParse(gens, os.Args[2])
        genB := genParse(gens, os.Args[3])
        if genA == -1 {
            logError("Generation " + os.Args[2] + " is invalid or does not exist")
            os.Exit(2)
        }
        if genB == -1 {
            logError("Generation " + os.Args[3] + " is invalid or does not exist")
            os.Exit(2)
        }

        handler := ""
        if len(os.Args) == 5 {
            handler = os.Args[4]
        }

        hasDiff := false
        for _, h := range config.Handlers {
            if handler != "" && h.Name != handler {
                continue
            }
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
            logInfo("Generations differ")
            os.Exit(1)
        } else {
            logInfo("Generations are identical")
            os.Exit(0)
        }
    } else if os.Args[1] == "switch" {
        if len(os.Args) < 3 {
            logUsage("eugene switch <targetGen> [--dry-run]")
        }

        targetGen := genParse(gens, os.Args[2])
        if targetGen == -1 {
            logError("The target generation is invalid or does not exist")
            os.Exit(1)
        }
        if targetGen == genGetCurrent(gens) {
            logError("Switching to the current generation makes no sense")
            os.Exit(1)
        }

        dryRun := hasFlag(os.Args, "--dry-run", 3)

        if doSwitch(config, gens, targetGen, dryRun) {
            os.Exit(0)
        } else {
            os.Exit(1)
        }
    } else if os.Args[1] == "show" {
        if len(os.Args) < 3 {
            logUsage("eugene show <gen> [handler]")
        }
        num := genParse(gens, os.Args[2])
        if num == -1 {
            logError("Generation '" + os.Args[2] + "' is invalid or does not exist")
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
        dryRun := hasFlag(os.Args, "--dry-run", 2)
        doUpgrade(config, dryRun)
    } else if os.Args[1] == "apply" {
        dryRun := hasFlag(os.Args, "--dry-run", 2)
        if doBuild(make([]string, 0), repo, gens, config) {
            latestGen := genGetLatest(gens)
            logInfo("Switching to newly built generation")
            doSwitch(config, gens, latestGen, dryRun)
        } else {
            logInfo("Switch canceled")
        }
    } else if os.Args[1] == "align" {
        dryRun := hasFlag(os.Args, "--dry-run", 2)
        doAlign(gens, dryRun)
    } else if os.Args[1] == "deletedups" {
        dryRun := hasFlag(os.Args, "--dry-run", 2)
        doDeleteDups(gens, dryRun)
        logInfo("Duplicate generations deleted")
        if hasFlag(os.Args, "--align", 2) {
            logInfo("Now aligning")
            doAlign(gens, dryRun)
        }
    } else if os.Args[1] == "rollback" {
        n := 1
        if len(os.Args) >= 3 {
            num, err := strconv.Atoi(os.Args[2])
            if err != nil {
                logError(os.Args[2] + " is an invalid number for parameter `n`")
            }
            n = num
        }
        dryRun := hasFlag(os.Args, "--dry-run", 3)
        if doRollback(config, gens, n, dryRun) {
            logAction("Rolled back " + os.Args[2] + " generations", dryRun)
        } else {
            os.Exit(1)
        }
    } else if os.Args[1] == "repair" {
        dryRun := hasFlag(os.Args, "--dry-run", 2)

        if doRepair(config, gens, dryRun) {
            os.Exit(0)
        } else {
            os.Exit(1)
        }        
    } else if os.Args[1] == "storage" {
        if os.Args[2] == "put" {
            if len(os.Args) < 6 {
                logUsage("eugene storage put <numGen> <namespace> <key> <value>")
                logUsage("echo value | eugene storage put <numGen> <namespace> <key>")
            }
            gen := genParse(gens, os.Args[3])
            if gen == -1 {
                logError("Generation " + os.Args[3] + " is invalid or does not exist")
                os.Exit(1)
            }
            ns := os.Args[4]
            key := os.Args[5]
            ok := false
            if len(os.Args) == 7 {
                ok = genStoragePut(gens, gen, ns, key, []string{os.Args[6]})
            } else {
                scanner := bufio.NewScanner(os.Stdin)
                var value []string
                for scanner.Scan() {
                    value = append(value, scanner.Text())
                }
                ok = genStoragePut(gens, gen, ns, key, value)
            }
            if ! ok {
                logError("Error writing value")
                os.Exit(1)
            }
        } else if os.Args[2] == "get" {
            if len(os.Args) != 6 {
                logUsage("eugene storage get <numGen> <namespace> <key>")
            }
            gen := genParse(gens, os.Args[3])
            if gen == -1 {
                logError("Generation " + os.Args[3] + " is invalid or does not exist")
                os.Exit(1)
            }
            ns := os.Args[4]
            key := os.Args[5]
            for _, val := range genStorageGet(gens, gen, ns, key) {
                fmt.Println(val)
            }
        } else {
            logUsage("eugene storage put <numGen> <namespace> <key> <value>")
            logUsage("eugene storage get <numGen> <namespace> <key>")
        }
    } else {
        logError("Unknown subcommand '" + os.Args[1] + "'")
        os.Exit(2)
    }
}
