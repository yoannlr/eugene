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

// todo genDelete = uniquement suppression
// bouger les messages dans doDelete
func genDelete(gens string, num int) bool {
    if num == 0 {
        logError("Deleting generation 0 is forbidden")
        return false
    }

    if num == genGetCurrent(gens) {
        logError("Deleting the current generation is forbidden")
        return false
    }

    if ! genExists(gens, num) {
        logError("Generation " + strconv.Itoa(num) + " does not exist")
        return false
    }
    
    if num == genGetLatest(gens) {
        prevGen := num - 1
        for ! genExists(gens, prevGen) {
            prevGen = prevGen - 1
            // on tombe sur la generation 0 au bout d'un moment
        }
        genSetLatest(gens, prevGen)
        logInfo("The latest generation is now " + strconv.Itoa(prevGen))
    }

    os.RemoveAll(genGetPath(gens, num))
    logInfo("Deleted generation " + strconv.Itoa(num))

    return true
}

func genGetPath(gens string, num int) string {
    if genExists(gens, num) {
        return filepath.Join(gens, strconv.Itoa(num))
    }
    return ""
}

func genSwitch(config Config, gens string, targetGen int, fromGen int, dryRun bool) bool {
    // variables d'environnement pour utilisation dans scripts
    os.Setenv("EUGENE_CURRENT_GEN", strconv.Itoa(fromGen))
    os.Setenv("EUGENE_TARGET_GEN", strconv.Itoa(targetGen))
    repair := (fromGen == 0)
    for _, h := range config.Handlers {
        os.Setenv("EUGENE_HANDLER_NAME", h.Name)
        if ! handlerSetup(h, gens, dryRun, repair) {
            return false
        }
        if ! handlerPreSwitch(h, dryRun) {
            return false
        }
        if ! handlerSync(h, dryRun) {
            return false
        }
        add, remove := genDiff(gens, fromGen, targetGen, h)
        if ! handlerRemove(h, remove, dryRun) {
            return false
        }
        if ! handlerAdd(h, add, dryRun) {
            return false
        }
        if ! handlerPostSwitch(h, dryRun) {
            return false
        }
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

func genStoragePut(gens string, num int, namespace string, key string, value []string) bool {
    if num == 0 || ! genExists(gens, num) {
        return false
    }
    storagePath := filepath.Join(gens, strconv.Itoa(num), "storage", namespace)
    if ! fileExists(storagePath) {
        os.Mkdir(storagePath, os.ModePerm)
    }
    keyPath := filepath.Join(storagePath, key)
    if len(value) > 0 && value[0] != "" {
        keyFile, err := os.Create(keyPath)
        if err != nil {
            panic(err)
        }
        for _, val := range value {
            keyFile.WriteString(val + "\n")
        }
        keyFile.Close()
    } else {
        // vide => suppression
        if fileExists(keyPath) {
            os.Remove(keyPath)
            nsContent, _ := os.ReadDir(storagePath)
            if len(nsContent) == 0 {
                logInfo("Namespace " + namespace + " now empty, deleting from generation")
                os.Remove(storagePath)
            }
        }
    }
    return true
}

func genStorageGet(gens string, num int, namespace string, key string) []string {
    if num == 0 || ! genExists(gens, num) {
        return nil
    }
    keyPath := filepath.Join(gens, strconv.Itoa(num), "storage", namespace, key)
    if ! fileExists(keyPath) {
        return nil
    }
    var res []string
    keyFile, _ := os.Open(keyPath)
    scanner := bufio.NewScanner(keyFile)
    for scanner.Scan() {
        res = append(res, scanner.Text())
    }
    return res
}