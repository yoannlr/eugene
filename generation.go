package main

import (
    "os"
    "path/filepath"
    "strconv"
    "slices"
    "bufio"
    "regexp"
)

func genCreate(gens string, num int, comment string) string {
    thisGenDir := filepath.Join(gens, strconv.Itoa(num))
    os.Mkdir(thisGenDir, os.ModePerm)
    if comment != "" {
        commentFile, _ := os.Create(filepath.Join(thisGenDir, "_comment"))
        commentFile.WriteString(comment + "\n")
        commentFile.Close()
    }
    return thisGenDir
}

func genTag(gens string, num int, tag string) {
    currentDir, _ := os.Getwd()
    os.Chdir(gens)
    os.Remove(tag)
    os.Symlink(strconv.Itoa(num), tag)
    os.Chdir(currentDir)
}

func genGetTagged(gens string, tag string) int {
    g, _ := os.Readlink(filepath.Join(gens, tag))
    num, _ := strconv.Atoi(g)
    return num
}

func genSetCurrent(gens string, num int) {
    genTag(gens, num, "current")
}

func genSetLatest(gens string, num int) {
    genTag(gens, num, "latest")
}

func genGetCurrent(gens string) int {
    return genGetTagged(gens, "current")
}

func genGetLatest(gens string) int {
    return genGetTagged(gens, "latest")
}

func genExists(gens string, num int) bool {
    return fileExists(filepath.Join(gens, strconv.Itoa(num)))
}

func genDiff(gens string, a int, b int, h Handler) ([]string, []string) {
    var add []string
    var remove []string

    inGenA := handlerGetEntries(gens, a, h)
    inGenB := handlerGetEntries(gens, b, h)

    // in a and not in b => remove
    for _, entry := range inGenA {
        if !slices.Contains(inGenB, entry) {
            remove = append(remove, entry)
        }
    }

    // in b and not in a => add
    for _, entry := range inGenB {
        if !slices.Contains(inGenA, entry) {
            add = append(add, entry)
        }
    }

    return add, remove
}

func genParse(gens string, arg string) int {
    if arg == "current" {
        return genGetCurrent(gens)
    } else if arg == "latest" {
        return genGetLatest(gens)
    } else {
        num, err := strconv.Atoi(arg)
        if err != nil || ! genExists(gens, num) {
            return -1
        }
        return num
    }
}

func genGetComment(gens string, num int) string {
    if genExists(gens, num) {
        commentFilePath := filepath.Join(genGetPath(gens, num), "_comment")
        if fileExists(commentFilePath) {
            commentFile, _ := os.Open(commentFilePath)
            scanner := bufio.NewScanner(commentFile)
            comment := ""
            for scanner.Scan() {
                comment = scanner.Text()
                // on ne lit que la premiere ligne
                break
            }
            return string(comment)
        }
    }
    return ""
}

func genDelete(gens string, num int) bool {
    if num == 0 {
        eugeneError("Deleting generation 0 is forbidden")
        return false
    }

    if num == genGetCurrent(gens) {
        eugeneError("Deleting the current generation is forbidden")
        return false
    }

    if ! genExists(gens, num) {
        eugeneError("Generation " + strconv.Itoa(num) + " does not exist")
        return false
    }
    
    if num == genGetLatest(gens) {
        prevGen := num - 1
        for ! genExists(gens, prevGen) {
            prevGen = prevGen - 1
            // on tombe sur la generation 0 au bout d'un moment
        }
        genSetLatest(gens, prevGen)
        eugeneMessage("The latest generation is now " + strconv.Itoa(prevGen))
    }

    os.RemoveAll(genGetPath(gens, num))
    eugeneMessage("Deleted generation " + strconv.Itoa(num))

    return true
}

func genGetPath(gens string, num int) string {
    if genExists(gens, num) {
        return filepath.Join(gens, strconv.Itoa(num))
    }
    return ""
}

func genSwitch(config Config, gens string, targetGen int, dryRun bool) bool {
    // variables d'environnement pour utilisation dans scripts
    os.Setenv("EUGENE_CURRENT_GEN", strconv.Itoa(genGetCurrent(gens)))
    os.Setenv("EUGENE_TARGET_GEN", strconv.Itoa(targetGen))
    for _, h := range config.Handlers {
        os.Setenv("EUGENE_HANDLER_NAME", h.Name)
        if ! handlerSetup(h, gens, dryRun) {
            return false
        }
        handlerHook(h, "before_switch", dryRun)
        if ! handlerSync(h, dryRun) {
            return false
        }
        add, remove := genDiff(gens, genGetCurrent(gens), targetGen, h)
        if ! handlerRemove(h, remove, dryRun) {
            return false
        }
        if ! handlerAdd(h, add, dryRun) {
            return false
        }
        if ! handlerHook(h, "after_switch", dryRun) {
            return false
        }
        eugeneMessage("")
    }

    if ! dryRun {
        genSetCurrent(gens, targetGen)
    }

    return true
}

func genGetAll(gens string) []int {
    generationRegex, _ := regexp.Compile("^[0-9]+$")
    var resultArr []int

    allGens, _ := os.ReadDir(gens)
    for _, g := range allGens {
        if generationRegex.MatchString(g.Name()) {
            num, _ := strconv.Atoi(g.Name())
            resultArr = append(resultArr, num)
        }
    }

    slices.Sort(resultArr)
    return resultArr
}

func genRenumber(gens string, old int, new int) {
    os.Rename(filepath.Join(gens, strconv.Itoa(old)), filepath.Join(gens, strconv.Itoa(new)))
}

func genGetHash(gens string, num int) string {
    if num == 0 {
        // todo gen 0 n'a pas de hash
        return ""
    }
    if ! genExists(gens, num) {
        return ""
    }
    hashFile, err := os.Open(filepath.Join(gens, strconv.Itoa(num), "_hash"))
    if err != nil {
        return ""
    }

    hash := ""
    scanner := bufio.NewScanner(hashFile)
    for scanner.Scan() {
        hash = scanner.Text()
        // hash sur la premiere ligne uniquement
        break
    }
    hashFile.Close()

    //eugeneMessage("Hash for generation " + strconv.Itoa(num) + " is " + hash)

    return hash
}