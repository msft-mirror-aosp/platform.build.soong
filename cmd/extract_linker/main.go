// Copyright 2017 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This tool extracts ELF LOAD segments from our linker binary, and produces an
// assembly file and linker script which will embed those segments as sections
// in another binary.
package main

import (
	"bytes"
	"debug/elf"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func main() {
	var asmPath string
	var scriptPath string

	flag.StringVar(&asmPath, "s", "", "Path to save the assembly file")
	flag.StringVar(&scriptPath, "T", "", "Path to save the linker script")
	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Error opening %q: %v", flag.Arg(0), err)
	}
	defer f.Close()

	ef, err := elf.NewFile(f)
	if err != nil {
		log.Fatalf("Unable to read elf file: %v", err)
	}

	asm := &bytes.Buffer{}
	script := &bytes.Buffer{}
	baseLoadAddr := uint64(0x1000)
	load := 0

	fmt.Fprintln(asm, ".globl __dlwrap_linker_offset")
	fmt.Fprintf(asm, ".set __dlwrap_linker_offset, 0x%x\n", baseLoadAddr)

	fmt.Fprintln(script, "ENTRY(__dlwrap__start)")
	fmt.Fprintln(script, "SECTIONS {")

	progsWithFlagsCount := make(map[string]int)

	for _, prog := range ef.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}

		progName := progNameFromFlags(prog.Flags, progsWithFlagsCount)
		sectionName := ".linker_" + progName
		symName := "__dlwrap_linker_" + progName

		flags := ""
		if prog.Flags&elf.PF_W != 0 {
			flags += "w"
		}
		if prog.Flags&elf.PF_X != 0 {
			flags += "x"
		}
		fmt.Fprintf(asm, ".section %s, \"a%s\"\n", sectionName, flags)

		if load == 0 {
			fmt.Fprintln(asm, ".globl __dlwrap_linker")
			fmt.Fprintln(asm, "__dlwrap_linker:")
			fmt.Fprintln(asm)
		}

		fmt.Fprintf(asm, ".globl %s\n%s:\n\n", symName, symName)

		fmt.Fprintf(script, "  %s 0x%x : {\n", sectionName, baseLoadAddr+prog.Vaddr)
		fmt.Fprintf(script, "    KEEP(*(%s));\n", sectionName)
		fmt.Fprintln(script, "  }")

		buffer, _ := ioutil.ReadAll(prog.Open())
		bytesToAsm(asm, buffer)

		// Fill in zeros for any BSS sections. It would be nice to keep
		// this as a true BSS, but ld/gold isn't preserving those,
		// instead combining the segments with the following segment,
		// and BSS only exists at the end of a LOAD segment.  The
		// linker doesn't use a lot of BSS, so this isn't a huge
		// problem.
		if prog.Memsz > prog.Filesz {
			fmt.Fprintf(asm, ".fill 0x%x, 1, 0\n", prog.Memsz-prog.Filesz)
		}
		fmt.Fprintln(asm)

		load += 1
	}

	fmt.Fprintln(asm, ".globl __dlwrap_linker_end")
	fmt.Fprintln(asm, "__dlwrap_linker_end:")
	fmt.Fprintln(asm)

	fmt.Fprintln(asm, `.section .note.android.embedded_linker,"a",%note`)

	fmt.Fprintln(script, "}")
	fmt.Fprintln(script, "INSERT BEFORE .note.android.embedded_linker;")

	if asmPath != "" {
		if err := ioutil.WriteFile(asmPath, asm.Bytes(), 0777); err != nil {
			log.Fatalf("Unable to write %q: %v", asmPath, err)
		}
	}

	if scriptPath != "" {
		if err := ioutil.WriteFile(scriptPath, script.Bytes(), 0777); err != nil {
			log.Fatalf("Unable to write %q: %v", scriptPath, err)
		}
	}
}

func bytesToAsm(asm io.Writer, buf []byte) {
	for i, b := range buf {
		if i%64 == 0 {
			if i != 0 {
				fmt.Fprint(asm, "\n")
			}
			fmt.Fprint(asm, ".byte ")
		} else {
			fmt.Fprint(asm, ",")
		}
		fmt.Fprintf(asm, "%d", b)
	}
	fmt.Fprintln(asm)
}

func progNameFromFlags(flags elf.ProgFlag, progsWithFlagsCount map[string]int) string {
	s := ""
	if flags&elf.PF_R != 0 {
		s += "r"
	}
	if flags&elf.PF_W != 0 {
		s += "w"
	}
	if flags&elf.PF_X != 0 {
		s += "x"
	}

	count := progsWithFlagsCount[s]
	count++
	progsWithFlagsCount[s] = count

	if count > 1 {
		s += strconv.Itoa(count)
	}

	return s
}
