package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"io/fs"
	"os"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/AJRDRGZ/fileinfo"
	"golang.org/x/exp/constraints"
)

func main() {
	// filter patten
	flagPattern := flag.String("p", "", "filter by pattern")
	flagAll := flag.Bool("a", false, "all files including hide files")
	flagNumberRecords := flag.Int("n", 0, "number of records")

	// order flags
	hasOrderByTime := flag.Bool("t", false, "sort by create at, oldest first")
	hasOrderBySize := flag.Bool("s", false, "sort by file size, smallest first")
	hasOrderReverse := flag.Bool("r", false, "reverse order while sorting")
	// Mapeamos los flags para que se puedan almacenar en las variables y tener acceso a los valores
	flag.Parse()

	path := flag.Arg(0)
	if path == "" {
		path = "."
	}
	dirs, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	fs := []file{}

	for _, dir := range dirs {
		isHidden := isHidden(dir.Name(), path)
		if isHidden && !*flagAll {
			continue
		}

		if *flagPattern != "" {
			isMatches, err := regexp.MatchString("(?i)"+*flagPattern, dir.Name())
			if err != nil {
				panic(err)
			}
			if !isMatches {
				continue
			}
		}

		f, err := getFile(dir, isHidden)
		if err != nil {
			panic(err)
		}
		fs = append(fs, f)
	}

	if !*hasOrderBySize || !*hasOrderByTime {
		orderByName(fs, *hasOrderReverse)
	}

	if *hasOrderBySize || !*hasOrderByTime {
		orderBySize(fs, *hasOrderReverse)
	}

	if *hasOrderByTime {
		orderByTime(fs, *hasOrderReverse)
	}

	// En caso de que -n sea cero o sea mayor a la cantidad de archivos a listar
	// lo que hacemos es configurar el valor de -n a la cantidad de archivos a listar.
	if *flagNumberRecords == 0 || *flagNumberRecords > len(fs) {
		*flagNumberRecords = len(fs)
	}

	printList(fs, *flagNumberRecords)
}

func getFile(dir fs.DirEntry, isHidden bool) (file, error) {
	info, err := dir.Info()
	if err != nil {
		return file{}, fmt.Errorf("dir.Info(): %w", err)
	}
	useName, groupName := fileinfo.GetUserAndGroup(info.Sys())
	f := file{
		name:             dir.Name(),
		isDir:            dir.IsDir(),
		isHidden:         isHidden,
		userName:         useName,
		groupName:        groupName,
		size:             int(info.Size()),
		modificationTime: info.ModTime(),
		mode:             info.Mode().String(),
	}
	setFileType(&f)
	return f, nil
}

func setFileType(f *file) {
	switch {
	case f.isDir:
		f.fileType = fileDirectory
	case isLink(*f):
		f.fileType = fileLink
	case isExec(*f):
		f.fileType = fileExecutable
	case isCompress(*f):
		f.fileType = fileCompress
	case isImage(*f):
		f.fileType = fileImage
	default:
		f.fileType = fileRegular
	}
}

func isLink(f file) bool {
	return strings.HasPrefix(strings.ToUpper(f.mode), "L")
}

func isExec(f file) bool {
	if runtime.GOOS == Windows {
		return strings.HasSuffix(f.name, exe)
	}
	return strings.Contains(f.mode, "x")
}

func isCompress(f file) bool {
	return strings.HasSuffix(f.name, zip) ||
		strings.HasSuffix(f.name, gz) ||
		strings.HasSuffix(f.name, tar) ||
		strings.HasSuffix(f.name, rar) ||
		strings.HasSuffix(f.name, deb)
}

func isImage(f file) bool {
	return strings.HasSuffix(f.name, png) ||
		strings.HasSuffix(f.name, jpg) ||
		strings.HasSuffix(f.name, jpeg) ||
		strings.HasSuffix(f.name, gif) ||
		strings.HasSuffix(f.name, webp)
}

func isHidden(fileName, basePath string) bool {
	filePath := fileName

	if runtime.GOOS == Windows {
		filePath = path.Join(basePath, fileName)
	}
	return fileinfo.IsHidden(filePath)
}

func markHidden(isHidden bool) string {
	if !isHidden {
		return ""
	}
	return yellow("ø")
}

func sortBy[T constraints.Ordered](i, j T, isReverse bool) bool {
	if isReverse {
		return i > j
	}
	return i < j
}

func orderByName(files []file, isReverse bool) {
	sort.SliceStable(files, func(i, j int) bool {
		return sortBy[string](strings.ToLower(files[i].name), strings.ToLower(files[j].name), isReverse)
	})
}

func orderBySize(files []file, isReverse bool) {
	sort.SliceStable(files, func(i, j int) bool {
		return sortBy[int](files[i].size, files[j].size, isReverse)
	})
}

func orderByTime(files []file, isReverse bool) {
	sort.SliceStable(files, func(i, j int) bool {
		return sortBy[int64](files[i].modificationTime.Unix(), files[j].modificationTime.Unix(), isReverse)
	})
}

func setColor(nameFile string, styleColor color.Attribute) string {
	switch styleColor {
	case color.FgBlue:
		return blue(nameFile)
	case color.FgGreen:
		return green(nameFile)
	case color.FgRed:
		return red(nameFile)
	case color.FgMagenta:
		return magenta(nameFile)
	case color.FgCyan:
		return cyan(nameFile)
	default:
		return nameFile
	}
}

func printList(fs []file, nRecords int) {
	for _, file := range fs[:nRecords] {
		style := mapStyleByFileType[file.fileType]
		fmt.Printf("%s %s %s %10d %s %s %s%s %s\n",
			file.mode,
			file.userName,
			file.groupName,
			file.size,
			file.modificationTime.Format("Jan  _2 15:04"),
			style.icon,
			setColor(file.name, style.color),
			style.symbol,
			markHidden(file.isHidden),
		)
	}
}
