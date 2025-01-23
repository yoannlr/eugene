package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"regexp"
	"slices"
	"strings"
	"bufio"
	"os/exec"

	"gopkg.in/yaml.v2"
)

const colorReset = "\033[0m"
const colorCyan = "\033[1;36m"
const colorGreen = "\033[0;32m"
const colorRed = "\033[0;31m"
const colorYellow = "\033[0;33m"
const colorPurple = "\033[1;35m"
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
	fmt.Println(colorGreen + "Info | " + colorReset + msg)
}

func eugeneError(msg string) {
	fmt.Println(colorRed + "Error | " + msg + colorReset)
}

func eugeneDebug(msg string) {
	fmt.Println("Debug | " + msg)
}

func eugeneUsage(showType bool, msg string) {
	if showType {
		fmt.Println(colorRed + "Usage | " + colorReset + msg)
	} else {
		fmt.Println(colorRed + "      | " + colorReset + msg)
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
		repo, _ = os.UserHomeDir()
		repo = filepath.Join(repo, ".eugene")
		os.Setenv("EUGENE_REPO", repo)
	}
	if ! fileExists(repo) {
		os.Mkdir(repo, os.ModePerm)
		eugeneMessage("Initialized new repository to " + repo)
		configInit(repo)
	}

	gens := os.Getenv("EUGENE_GENS")
	if gens == "" {
		gens = filepath.Join(repo, ".gens")
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
		eugeneUsage(true, "eugene <build|list|diff|switch|delete|show>")
		eugeneUsage(false, "")
		eugeneUsage(false, textBold + "eugene build [comment]" + colorReset)
		eugeneUsage(false, "  creates a new generation with the current file")
		eugeneUsage(false, "  the generation is automatically removed if it does not differ from the latest")
		eugeneUsage(false, textBold + "eugene list" + colorReset)
		eugeneUsage(false, "  lists all the generations")
		eugeneUsage(false, "  the current one is indicated with an arrow")
		eugeneUsage(false, textBold + "eugene diff <fromGen> <toGen>" + colorReset)
		eugeneUsage(false, "  shows the difference between two generations")
		eugeneUsage(false, "  exit code is 0 if they're the same, 1 if they're different")
		eugeneUsage(false, textBold + "eugene switch <targetGen> [--dry-run]" + colorReset)
		eugeneUsage(false, "  switches to the target generation (sync/remove/install packages)")
		eugeneUsage(false, textBold + "eugene delete <gen1> [gen2 gen3 ...]" + colorReset)
		eugeneUsage(false, "  deletes one or more generations")
		eugeneUsage(false, "  deletion of the current generation and generation 0 is forbidden")
		eugeneUsage(false, textBold + "eugene show <gen> [handler]" + colorReset)
		eugeneUsage(false, "  show all the entries in a generation")
		eugeneUsage(false, textBold + "eugene upgrade [--dry-run]" + colorReset)
		eugeneUsage(false, "  perform upgrade command of each handler")
		eugeneUsage(false, "")
		eugeneUsage(false, "eugene can be configured with the following environment variables")
		eugeneUsage(false, "  - EUGENE_REPO - list of entries for each handler, defaults to ~/.eugene")
		eugeneUsage(false, "  - EUGENE_GENS - internal storage for generations, defaults to repo/.gens")
		eugeneUsage(false, "  - EUGENE_HOOKS - hook scripts for each handler, defaults to repo/hooks")
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
		generationRegex, _ := regexp.Compile("^[0-9]+$")
		allGens, _ := os.ReadDir(gens)
		currentGen := genGetCurrent(gens)
		for _, g := range allGens {
			if g.Name() == "latest" || g.Name() == "current" {
				continue
			}

			if ! generationRegex.MatchString(g.Name()) {
				continue
			}

			num, _ := strconv.Atoi(g.Name())
			if num == currentGen {
				fmt.Print(colorGreen + "->")
			} else {
				fmt.Print("  ")
			}

			fmt.Print(" " + g.Name() + " ")

			comment := genGetComment(gens, num)

			if comment != "" {
				fmt.Print(string(comment))
			} else {
				fmt.Print("(no comment)")
			}

			fmt.Print(colorReset + "\n")
		}
	} else if os.Args[1] == "build" {
		newGen := genGetLatest(gens) + 1
		comment := ""
		if len(os.Args) > 1 {
			comment = strings.Join(os.Args[2:], " ")
		}
		newGenDir := genCreate(gens, newGen, comment)

		commentRegex, _ := regexp.Compile("^#")
		emptyLineRegex, _ := regexp.Compile("^$")

		hasDiff := true

		for _, h := range config.Handlers {
			handlerStatus(h, "build")

			// trouver les fichiers a inclure
			filesRegex, _ := regexp.Compile(fmt.Sprintf("^%s.*$", h.Name))
			repoFiles, _ := os.ReadDir(repo)

			var handlerFiles []string
			for _, f := range repoFiles {
				if filesRegex.MatchString(f.Name()) {
					eugeneMessage("+ include file " + f.Name())
					handlerFiles = append(handlerFiles, f.Name())
				}
			}

			// generer le resultat
			if len(handlerFiles) > 0 {
				var handlerEntries []string
				for _, f := range handlerFiles {
					file, _ := os.Open(f)
					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						line := scanner.Text()
						if ! emptyLineRegex.MatchString(line) && ! commentRegex.MatchString(line) {
							handlerEntries = append(handlerEntries, line)
						}
					}
					file.Close()
				}

				slices.Sort(handlerEntries) // sort
				handlerEntries = slices.Compact(handlerEntries) // uniq

				handlerResult, _ := os.Create(filepath.Join(newGenDir, h.Name))
				for _, p := range handlerEntries {
					handlerResult.WriteString(p + "\n")
				}
				handlerResult.Close()

				if ! hasDiff {
					add, remove := genDiff(gens, genGetLatest(gens), newGen, h)
					if len(add) > 0 || len(remove) > 0 {
						hasDiff = true
					}
				}
			}
		}

		if hasDiff {
			genSetLatest(gens, newGen)
			eugeneMessage(colorGreen + "Done building generation " + strconv.Itoa(newGen) + colorReset)
		} else {
			genDelete(gens, newGen)
			eugeneMessage("No difference with the latest generation, build removed")
		}
	} else if os.Args[1] == "delete" {
		if len(os.Args) > 1 {
			deleteGens := os.Args[2:]
			// les mettre dans l'ordre permet de ne faire qu'un seul changement du pointeur latest (si besoin)
			slices.Sort(deleteGens)
			for _, g := range deleteGens {
				num := genParse(gens, g)
				if ! genDelete(gens, num) {
					break
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
		if genA == -1 {
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
				fmt.Println(colorRed + "- " + strings.Join(remove, " ") + colorReset)
				fmt.Println(colorGreen + "+ " + strings.Join(add, " ") + colorReset)
			} else {
				for _, entry := range remove {
					fmt.Println(colorRed + "- " + entry + colorReset)
				}
				for _, entry := range add {
					fmt.Println(colorGreen + "+ " + entry + colorReset)
				}
			}
		}

		if hasDiff {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	} else if os.Args[1] == "switch" {
		if len(os.Args) < 3 {
			eugeneUsage(true, "eugene switch <targetGen> [--dry-run]")
			os.Exit(1)
		}

		targetGen := genParse(gens, os.Args[2])
		if targetGen == -1 {
			eugeneError("The target generation is invalid")
			os.Exit(1)
		}
		if targetGen == genGetCurrent(gens) {
			eugeneError("Switching to the current generation makes no sense")
			os.Exit(1)
		}

		dryRun := false
		if len(os.Args) == 4 && os.Args[3] == "--dry-run" {
			dryRun = true
		}

		if genSwitch(config, gens, targetGen, dryRun) {
			if dryRun {
				eugeneMessage("Switched to generation " + strconv.Itoa(targetGen) + colorYellow + " (dry run)" + colorReset)
			} else {
				eugeneMessage("Switched to generation " + strconv.Itoa(targetGen))
			}
			os.Exit(0)
		} else {
			eugeneError("Switch to generation " + strconv.Itoa(targetGen) + " failed")
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
		dryRun := false
		if len(os.Args) == 3 && os.Args[2] == "--dry-run" {
			dryRun = true
		}
		eugeneMessage("Running upgrade")
		for _, h := range config.Handlers {
			handlerStatus(h, "upgrade")
			if h.Upgrade == "" {
				handlerMessage(h, "Command undefined")
			} else {
				handlerExec(h.Upgrade, dryRun)
			}
		}
	}
}
