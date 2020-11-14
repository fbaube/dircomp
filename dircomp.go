package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"

	FU "github.com/fbaube/fileutils"
)

var dirM = ""
var dirS = ""
var execute = true

const eg = "/dirpath"

// FilesReverseSortedBySize is used for sorting.
type FilesReverseSortedBySize []os.FileInfo

func init() {
	flag.StringVar(&dirM, "m", eg, "master dir (files aren't deleted)")
	flag.StringVar(&dirS, "s", eg, "slave  dir (dupes can be deleted)")
	flag.BoolVar(&execute, "x", false, "execute (prompt for) deletions interactively - not just a dry run")
}

func main() {
	nArgs := 2
	flag.Parse()
	if flag.NFlag() == 0 && flag.NArg() == 0 {
		fmt.Println("dircomp: process duplicate files in a slave")
		fmt.Println("         directory vis-à-vis a master directory.")
		fmt.Println("       - Checks file names, sizes, and contents.")
		fmt.Println("       - Open up two file explorer windows to")
		fmt.Println("         see its amazing effects interactively.")
		fmt.Println("\"dircomp -h\" for help.")
		os.Exit(0)
	}
	if dirM == eg {
		dirM = "Not specified"
		nArgs--
	}
	if dirS == eg {
		dirS = "Not specified"
		nArgs--
	}
	fmt.Println("Master dir:", dirM)
	fmt.Println(" Slave dir:", dirS)
	if execute {
		fmt.Println("   ==> EXECUTE")
	}

	// fmt.Println("nArgs", flag.NArg(), nArgs)
	if nArgs == 0 && flag.NArg() == 2 {
		dirM = flag.Arg(0)
		dirS = flag.Arg(1)
		fmt.Println("Making assumptions!")
		fmt.Println("Master dir:", dirM)
		fmt.Println(" Slave dir:", dirS)
	} else if nArgs != 2 || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "FAIL")
		os.Exit(1)
	}
	fM := FU.Must(FU.OpenDir(dirM))
	nM, fisM := FU.GetDirFiles(fM)
	fmt.Println("Found", nM, "files in", dirM)
	fS := FU.Must(FU.OpenDir(dirS))
	nS, fisS := FU.GetDirFiles(fS)
	fmt.Println("Found", nS, "files in", dirS)
	if nM == 0 || nS == 0 {
		fmt.Println("Nothing to do")
		os.Exit(1)
	}
	// disable input buffering
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	defer exec.Command("stty", "-F", "/dev/tty", "icanon", "min", "1").Run()
	pressAnyKey()

	dedupeByNames(dirM, fisM, dirS, fisS)
	dedupeByLengths(dirM, fisM, dirS, fisS)
}

func pressAnyKey() {
	fmt.Printf("Press any key to continue...")
	var b = make([]byte, 1)
	os.Stdin.Read(b)
	fmt.Println("")
}

// dumpFIs
func dumpFIs(aPfx string, aFIs []os.FileInfo) {
	for i := range aFIs {
		fi := aFIs[i]
		if fi != nil {
			fmt.Printf("\t %s[%v]: <%v:%s> \n", aPfx, i, fi.Size(), fi.Name())
		} else {
			fmt.Printf("\t %s[%v]: <--> \n", aPfx, i)
		}
	}
}

// dedupeByNames
func dedupeByNames(dirM string, fisM []os.FileInfo, dirS string, fisS []os.FileInfo) {
	dumpFIs("M", fisM)
	dumpFIs("s", fisS)
	iM, iS := 0, 0
	for iM < len(fisM) && iS < len(fisS) {
		// fmt.Println("Loop entry", iM, iS)
		for fisM[iM] == nil {
			iM++
			fmt.Println("M: skip")
		}
		for fisS[iS] == nil {
			iS++
			fmt.Println("s: skip")
		}
		// fmt.Println("Files at", iM, iS)
		fiM := fisM[iM]
		fiS := fisS[iS]
		nameM := fiM.Name()
		nameS := fiS.Name()
		if nameM > nameS {
			iS++
			continue
		}
		if nameM < nameS {
			iM++
			continue
		}
		if nameM != nameS {
			panic("OUCH")
		}
		if os.SameFile(fiM, fiS) {
			panic(nameM)
		}
		if fiM.Size() != fiS.Size() {
			fmt.Printf("[%v,%v]: <%v,%v:%s> \t same name, diff size \n",
				iM, iS, fiM.Size(), fiS.Size(), nameM)
		} else {
			// Gotta rebuild the file names and open them !
			fnamM := path.Join(dirM, nameM)
			fnamS := path.Join(dirS, nameS)
			fM := FU.Must(FU.OpenRO(fnamM))
			fS := FU.Must(FU.OpenRO(fnamS))
			if FU.SameContents(fM, fS) {
				fmt.Printf("[%v,%v]:   <%v:%s> \t same name, same contents \n",
					iM, iS, fiM.Size(), nameM)
			} else {
				fmt.Printf("[%v,%v]:   <%v:%s> \t same name, same size, diff contents \n",
					iM, iS, fiM.Size(), nameM)
			}
		}
		iM++
		iS++
	}
}

// Len is for sorting.
func (DL FilesReverseSortedBySize) Len() int {
	return len(DL)
}

// Less is for sorting
func (DL FilesReverseSortedBySize) Less(i, j int) bool {
	if DL[i] == nil {
		return true
	}
	if DL[j] == nil {
		return false
	}
	return DL[i].Size() > DL[j].Size()
}

// Swap is for sorting.
func (DL FilesReverseSortedBySize) Swap(i, j int) {
	DL[i], DL[j] = DL[j], DL[i]
}

// dedupeInDir
func dedupeInDir(aDirPath string, aFIs []os.FileInfo) {
	for i := range aFIs {
		if i == 0 {
			continue
		}
		fi1 := aFIs[i-1]
		fi2 := aFIs[i]
		if fi1 == nil || fi2 == nil {
			continue
		}
		if fi1.Size() != fi2.Size() {
			continue
		}
		if fi1.Size() == 0 || fi2.Size() == 0 {
			continue
		}
		// Gotta rebuild the file names and open them !
		fnam1 := path.Join(aDirPath, fi1.Name())
		fnam2 := path.Join(aDirPath, fi2.Name())
		f1 := FU.Must(FU.OpenRO(fnam1))
		f2 := FU.Must(FU.OpenRO(fnam2))
		if !FU.SameContents(f1, f2) {
			continue
		}
		fmt.Println("Duplicate files in same directory:")
		fmt.Println("\t", fnam1)
		fmt.Println("\t", fnam2)
	}
}

// dedupeByLengths uses FilesReverseSortedBySize.
func dedupeByLengths(dirM string, fisM []os.FileInfo, dirS string, fisS []os.FileInfo) {
	sort.Sort(FilesReverseSortedBySize(fisM))
	sort.Sort(FilesReverseSortedBySize(fisS))
	dumpFIs("M", fisM)
	dumpFIs("s", fisS)
	// In each list, check for files that seem to be dupes with different names
	// (i.e. unintentional copies of each other)
	dedupeInDir(dirM, fisM)
	dedupeInDir(dirS, fisS)
	iM, iS := 0, 0
	for iM < len(fisM) && iS < len(fisS) {
		// fmt.Println("Loop entry", iM, iS)
		for fisM[iM] == nil {
			iM++
			fmt.Println("M: skip")
		}
		for fisS[iS] == nil {
			iS++
			fmt.Println("s: skip")
		}
		// fmt.Println("Files at", iM, iS)
		fiM := fisM[iM]
		fiS := fisS[iS]
		sizeM := fiM.Size()
		sizeS := fiS.Size()
		if sizeM < sizeS {
			iS++
			continue
		}
		if sizeM > sizeS {
			iM++
			continue
		}
		nameM := fiM.Name()
		nameS := fiS.Name()
		if sizeM != sizeS {
			panic(nameM)
		}
		if os.SameFile(fiM, fiS) {
			panic(nameM)
		}
		// Gotta rebuild the file names and open them !
		fnamM := path.Join(dirM, nameM)
		fnamS := path.Join(dirS, nameS)
		fM := FU.Must(FU.OpenRO(fnamM))
		fS := FU.Must(FU.OpenRO(fnamS))
		if FU.SameContents(fM, fS) {
			fmt.Printf("[%v,%v]:   <%v:M:%s:s:%s> \t same contents \n",
				iM, iS, sizeM, nameM, nameS)
		} else {
			fmt.Printf("[%v,%v]:   <%v:M:%s:s:%s> \t same size, diff contents \n",
				iM, iS, sizeM, nameM, nameS)
		}
		iM++
		iS++
	}
}
