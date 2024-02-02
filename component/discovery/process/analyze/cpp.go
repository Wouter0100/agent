package analyze

import (
	"strings"

	"github.com/xyproto/ainur"
)

const (
	LabelCPP      = "__meta_process_cpp__"
	LabelCompiler = "__meta_process_binary_compiler__"
	LabelStatic   = "__meta_process_binary_static__"
	LabelStripped = "__meta_process_binary_striped__"
)

func analyzeBinary(input Input, a *Results) error {
	m := a.Labels
	libs, err := input.ElfFile.ImportedLibraries()
	if err != nil {
		return err
	}

	for _, lib := range libs {
		if strings.Contains(lib, "libc++") || strings.Contains(lib, "libstdc++") {
			m[LabelCPP] = "true"
			break
		}
	}

	m[LabelCompiler] = ainur.Compiler(input.ElfFile)
	if ainur.Static(input.ElfFile) {
		m[LabelStatic] = "true"
	} else {
		m[LabelStatic] = "false"
	}
	if ainur.Stripped(input.ElfFile) {
		m[LabelStripped] = "true"
	} else {
		m[LabelStripped] = "false"
	}

	return nil
}
