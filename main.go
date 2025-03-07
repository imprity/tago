package main

import (
	"fmt"
	"log"
	"os"

	"crypto/md5"
	"crypto/sha256"

	"flag"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

var ErrLogger *log.Logger = log.New(os.Stderr, "ERROR: ", log.Lshortfile)
var WarnLogger *log.Logger = log.New(os.Stderr, "WARN: ", log.Lshortfile)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO: ", log.Lshortfile)

var (
	FlagCheckHash bool
)

func init() {
	flag.BoolVar(&FlagCheckHash, "c", false, "check file hash instead")
}

type TagoValue struct {
	Source string
	Value  string
}

func main() {
	flag.Parse()

	args := flag.Args()

	// if no argument is give,
	// print help and exit
	if len(args) <= 0 {
		flag.Usage()
		os.Exit(1)
	}

	var filePath = args[0]

	if FlagCheckHash {
		CheckHashMain(filePath)
	} else {
		TagoMain(filePath)
	}
}

func TagoMain(filePath string) {
	tagoFiles, err := findTagosForFile(filePath)

	// if we couldn't find any tago files
	// just exit normally
	if len(tagoFiles) <= 0 && err == nil {
		fmt.Printf("could not find any tago files")
		os.Exit(0)
	}

	// report errors
	if err != nil {
		if len(tagoFiles) <= 0 {
			ErrLogger.Fatalf("could not find any tago files: %s", err)
		} else {
			WarnLogger.Printf("error while finding tago files: %s", err)
		}
	}

	fmt.Printf("\n")

	// print key values from tago files
	{
		keyValue := make(map[string]TagoValue)
		for i := len(tagoFiles) - 1; i >= 0; i-- {
			filePath := tagoFiles[i]
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				WarnLogger.Printf("could not open %s: %s", filePath, err)
				continue
			}

			kv, err := parseTagoFile(filePath, fileContent)
			if err != nil {
				WarnLogger.Printf("could not parse %s: %s", filePath, err)
				continue
			}

			for k, v := range kv {
				keyValue[k] = v
			}
		}

		printKeyValue(keyValue)
	}
}

func CheckHashMain(filePath string) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		ErrLogger.Fatal("could not open %s: %s", filePath, err)
	}

	shaSum := sha256.Sum256(fileContent)
	md5Sum := md5.Sum(fileContent)

	fmt.Printf("\n")
	fmt.Printf("hashes:\n")
	fmt.Printf("    sha256: %x\n", shaSum)
	fmt.Printf("    md5   : %x\n", md5Sum)
}

func getNameAndExt(path string) (string, string) {
	path = filepath.Base(path)
	ext := filepath.Ext(path)
	name := path[0 : len(path)-len(ext)]
	return name, ext
}

func isPathSame(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)

	return a == b
}

func isTagoFile(path string) bool {
	ext := filepath.Ext(path)
	if strings.ToLower(ext) == ".tago" {
		return true
	}
	return false
}

func fileExistsAndRegular(path string) error {
	// check if file exists
	info, err := os.Stat(path)

	if err == nil { // file exists
		mode := info.Mode()
		if !mode.IsRegular() { // but it's not regular
			return fmt.Errorf("%s is not a regular file", path)
		}
		return nil
	}

	return err
}

// returns empty path if it couldn't find any
func findTagosForFile(filePath string) ([]string, error) {
	filePath = filepath.Clean(filePath) // why not

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if !(fileInfo.Mode().IsRegular() || fileInfo.Mode().IsDir()) {
		return nil, fmt.Errorf("%s is not a regular file nor directory")
	}

	var tagos []string

	// get abs
	fileAbsPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get absolute path of %s: %w",
			filePath, err,
		)
	}

	var fileDir, fileName, fileExt string

	if fileInfo.Mode().IsRegular() {
		fileDir = filepath.Dir(fileAbsPath)
		fileName, fileExt = getNameAndExt(filePath)
	} else {
		fileDir = fileAbsPath
		fileName = filepath.Base(filePath)
	}
	_ = fileExt

	dirents, err := os.ReadDir(fileDir)
	if err != nil {
		return nil, err
	}

	var fileTago string
	var rootTago string

	for _, dirent := range dirents {
		if !dirent.Type().IsRegular() {
			continue
		}

		direntPath := filepath.Join(fileDir, dirent.Name())

		// skip the same file
		if isPathSame(fileAbsPath, direntPath) {
			continue
		}
		// skip none tago file
		if !isTagoFile(dirent.Name()) {
			continue
		}

		name, _ := getNameAndExt(dirent.Name())

		// we found tago file with the same name
		if fileInfo.Mode().IsRegular() && name == fileName {
			fileTago = direntPath
		}

		// if we found tago.tago file
		if name == "tago" {
			rootTago = direntPath
		}
	}

	if fileInfo.Mode().IsRegular() && isTagoFile(fileAbsPath) {
		// if file it self is a tago file
		// add it to tagos
		fileTago = fileAbsPath
	}

	if len(fileTago) > 0 {
		tagos = append(tagos, fileTago)
	}
	if len(rootTago) > 0 {
		tagos = append(tagos, rootTago)
	}

	// start going upward to find more tago files
	curDir := fileDir
	for {
		prevCurDir := curDir
		curDir = filepath.Dir(prevCurDir)
		if curDir == prevCurDir || len(curDir) == 0 { // we have reached the top
			return tagos, nil
		}

		dirents, err = os.ReadDir(curDir)
		if err != nil {
			return tagos, err
		}

		for _, dirent := range dirents {
			if !dirent.Type().IsRegular() {
				continue
			}
			direntPath := filepath.Join(curDir, dirent.Name())

			// skip none tago file
			if !isTagoFile(dirent.Name()) {
				continue
			}

			name, _ := getNameAndExt(dirent.Name())

			// we found tago.tago file
			if name == "tago" {
				tagos = append(tagos, direntPath)
				break
			}
		}
	}

	return tagos, nil
}

func parseTagoFile(filePath string, fileContent []byte) (map[string]TagoValue, error) {
	if !utf8.Valid(fileContent) {
		return nil, fmt.Errorf("file is not a valid utf8")
	}

	text := string(fileContent)

	keyValue := make(map[string]TagoValue)

	text = strings.ReplaceAll(text, "\r\n", "\n")

	lines := strings.Split(text, "\n")

	var insideMultiline bool = false
	var multiLineKey string = ""
	var multiLineValue = ""

	for _, line := range lines {
		line := strings.TrimSpace(line)

		if insideMultiline {
			if line == "]" {
				keyValue[multiLineKey] = TagoValue{
					Source: filePath,
					Value:  multiLineValue,
				}

				insideMultiline = false
				multiLineKey = ""
				multiLineValue = ""
			} else {
				if len(multiLineValue) <= 0 {
					multiLineValue += line
				} else {
					multiLineValue += "\n" + line
				}
			}
		} else {
			if strings.HasPrefix(line, "//") { // skip comments
				continue
			}

			sepAt := strings.Index(line, ":")
			if sepAt >= 0 {
				key := strings.ToLower(strings.TrimSpace(line[0:sepAt]))

				valueLine := line[sepAt+1:]
				valueLine = strings.TrimSpace(valueLine)

				if valueLine == "[" {
					insideMultiline = true
					multiLineKey = key
					multiLineValue = ""
				} else {
					keyValue[key] = TagoValue{
						Source: filePath,
						Value:  valueLine,
					}
				}
			}
		}
	}

	return keyValue, nil
}

func printKeyValue(keyValue map[string]TagoValue) {
	keys := make([]string, len(keyValue))
	{
		i := 0
		for k := range keyValue {
			keys[i] = k
			i++
		}
	}

	slices.Sort(keys)

	for _, k := range keys {
		value := keyValue[k].Value

		if strings.Index(value, "\n") < 0 {
			fmt.Printf("%s: %s\n", k, value)
		} else {
			fmt.Printf("%s: [\n", k)

			lines := strings.Split(value, "\n")
			for _, line := range lines {
				fmt.Printf("    %s\n", line)
			}

			fmt.Printf("]\n")
		}
		fmt.Printf("    // \"%s\"\n", keyValue[k].Source)
		fmt.Printf("\n")
	}
}
