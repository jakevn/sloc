package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type lang struct {
	Name            string
	Filetypes       []string
	Comment         []string
	IgnoreIfOnly    []string
	FilenameIgnore  []string
	DirectoryIgnore []string
	TestFilename    []string
	TestDirname     []string
}

type count struct {
	LineCount int
	CharCount int
	FileCount int
}

var commonIgnore = []string{"}", "{", ")", "},", "),", "[", "]", "],"}

var languages = map[string]lang{
	"Python": {
		Filetypes:       []string{".py"},
		Comment:         []string{"#"},
		IgnoreIfOnly:    commonIgnore,
		TestFilename:    []string{"test_"},
		TestDirname:     []string{"test"},
		DirectoryIgnore: []string{"venv", "egg-info"},
	},
	"Go": {
		Filetypes:       []string{".go"},
		Comment:         []string{"//"},
		IgnoreIfOnly:    commonIgnore,
		DirectoryIgnore: []string{"vendor"},
		TestFilename:    []string{"_test"},
	},
	"Ruby": {
		Filetypes:       []string{".rb"},
		Comment:         []string{"#"},
		IgnoreIfOnly:    append(commonIgnore, "end"),
		DirectoryIgnore: []string{"vendor"},
		TestDirname:     []string{"spec"},
		TestFilename:    []string{"_spec"},
	},
	"C": {
		Filetypes:    []string{".c", ".h", ".cc"},
		Comment:      []string{"//"},
		IgnoreIfOnly: commonIgnore,
		TestDirname:  []string{"test"},
	},
	"C++": {
		Filetypes:    []string{".cpp", ".hpp"},
		Comment:      []string{"//"},
		IgnoreIfOnly: commonIgnore,
		TestDirname:  []string{"test"},
	},
	"C#": {
		Filetypes:    []string{".cs"},
		Comment:      []string{"//"},
		IgnoreIfOnly: commonIgnore,
		TestDirname:  []string{"test"},
	},
	"JavaScript": {
		Filetypes:    []string{".js"},
		Comment:      []string{"//"},
		IgnoreIfOnly: commonIgnore,
		TestDirname:  []string{"test"},
	},
	"JSON": {
		Filetypes:    []string{".json"},
		IgnoreIfOnly: commonIgnore,
	},
	"Java": {
		Filetypes:    []string{".java"},
		Comment:      []string{"//"},
		IgnoreIfOnly: commonIgnore,
		TestDirname:  []string{"test"},
	},
	"YAML": {
		Filetypes:    []string{".yml"},
		Comment:      []string{"#"},
		IgnoreIfOnly: commonIgnore,
	},
	"HTML": {
		Filetypes: []string{".html"},
	},
	"CSS": {
		Filetypes: []string{".css", ".scss"},
	},
	"Shell": {
		Filetypes:    []string{".sh"},
		Comment:      []string{"#"},
		IgnoreIfOnly: commonIgnore,
	},
	"Powershell": {
		Filetypes: []string{".ps1"},
	},
	"Rust": {
		Comment:      []string{"//"},
		IgnoreIfOnly: commonIgnore,
		Filetypes:    []string{".rs"},
		TestDirname:  []string{"test"},
	},
}

const maxFd = 10

var (
	noTest      = false
	onlyLang    = ""
	fdSem       = make(chan struct{}, maxFd)
	dirWg       = &sync.WaitGroup{}
	countWg     = &sync.WaitGroup{}
	counterLock = &sync.Mutex{}
	counters    = map[string]count{}
)

func main() {
	flag.BoolVar(&noTest, "notest", false, "ignore test files/directories")
	flag.StringVar(&onlyLang, "lang", "", "only count source for this language")
	flag.Parse()

	if onlyLang != "" {
		removeOtherLangs(onlyLang)
	}

	for langName := range languages {
		counters[langName] = count{}
	}

	startPath, err := filepath.Abs("")
	if err != nil {
		log.Fatal(err.Error())
	}

	dirWg.Add(1)
	countDir(startPath)
	dirWg.Wait()
	countWg.Wait()

	printResult()
}

func printResult() {
	totalLines := 0
	totalChars := 0
	totalFiles := 0
	for name, l := range counters {
		if l.LineCount == 0 {
			continue
		}

		fmt.Println(name,
			"\n  Lines:", commaInt(l.LineCount),
			"\n  Chars:", commaInt(l.CharCount),
			"\n  Files:", commaInt(l.FileCount),
			"\n  Chars/Line:", commaInt(l.CharCount/l.LineCount),
			"\n  Lines/File:", commaInt(l.LineCount/l.FileCount),
		)
		totalLines += l.LineCount
		totalChars += l.CharCount
		totalFiles += l.FileCount
	}
	fmt.Println("Total Lines:", commaInt(totalLines))
	fmt.Println("Total Chars:", commaInt(totalChars))
	fmt.Println("Total Files:", commaInt(totalFiles))
}

func commaInt(num int) string {
	sNum := strconv.Itoa(num)
	offset := len(sNum) % 3
	var res string
	for i, c := range sNum {
		if i > 0 && (i-offset)%3 == 0 {
			res += ","
		}
		res += string(c)
	}
	return res
}

func removeOtherLangs(keepLang string) {
	keepLang = strings.ToLower(keepLang)

	for langName := range languages {
		if strings.ToLower(langName) != keepLang {
			delete(languages, langName)
		}
	}
}

func countDir(dirPath string) {
	fdSem <- struct{}{}

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	<-fdSem

	for _, f := range files {
		filePath := path.Join(dirPath, f.Name())

		if f.IsDir() {
			dirWg.Add(1)
			countDir(filePath)
		} else {
			isSource, lang := isSourceFile(filePath)
			if !isSource || ignoreDir(dirPath, lang) || ignoreFile(f.Name(), lang) {
				continue
			}
			countWg.Add(1)
			go countFile(filePath, lang)
		}
	}

	dirWg.Done()
}

func countFile(path, langName string) {
	fdSem <- struct{}{}

	file, err := os.Open(path)
	if err != nil {
		log.Fatal("Can't open file:", err.Error())
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	lineCount := 0
	charCount := 0
	for scanner.Scan() {
		if ignoreLine(languages[langName], scanner.Text()) {
			continue
		}
		lineCount += 1
		charCount += len(scanner.Text())
	}

	file.Close()
	<-fdSem

	counterLock.Lock()
	l := counters[langName]
	l.LineCount += lineCount
	l.CharCount += charCount
	l.FileCount += 1
	counters[langName] = l
	counterLock.Unlock()

	countWg.Done()
}

func ignoreDir(dirPath, langName string) bool {
	for _, ignoreDir := range languages[langName].DirectoryIgnore {
		if strings.Contains(dirPath, ignoreDir) {
			return true
		}
	}

	if noTest {
		for _, ignoreFile := range languages[langName].TestDirname {
			if strings.Contains(dirPath, ignoreFile) {
				return true
			}
		}
	}
	return false
}

func isSourceFile(filePath string) (bool, string) {
	for langName, lang := range languages {
		for _, fileSuffix := range lang.Filetypes {
			if strings.HasSuffix(filePath, fileSuffix) {
				return true, langName
			}
		}
	}
	return false, ""
}

func ignoreFile(fileName, langName string) bool {
	for _, ignoreFile := range languages[langName].FilenameIgnore {
		if strings.Contains(fileName, ignoreFile) {
			return true
		}
	}

	if noTest {
		for _, ignoreFile := range languages[langName].TestFilename {
			if strings.Contains(fileName, ignoreFile) {
				return true
			}
		}
	}
	return false
}

func ignoreLine(l lang, line string) bool {
	return isEmpty(line) || isComment(l, line) || isIgnoreIfOnly(l, line)
}

func isIgnoreIfOnly(l lang, line string) bool {
	if len(l.IgnoreIfOnly) == 0 {
		return false
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}

	for _, ignoreIfOnly := range l.IgnoreIfOnly {
		if fields[0] == ignoreIfOnly || line == ignoreIfOnly {
			return true
		}
	}
	return false
}

func isEmpty(line string) bool {
	return len(strings.Fields(line)) == 0
}

func isComment(l lang, line string) bool {
	if len(l.Comment) == 0 {
		return false
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}

	for _, commentStart := range l.Comment {
		strings.HasPrefix(fields[0], commentStart)
	}
	return false
}
