package commands

import (
	"flag"
	"regexp"
	"strings"
)

// negativeNumber matches values argparse accepts after an option even though
// they start with a dash, e.g. --max -1.
var negativeNumber = regexp.MustCompile(`^-\d+$|^-\d*\.\d+$`)

// parseFlags parses flags that may appear anywhere in args (before or after
// positionals), matching Python argparse behavior. Go's flag package stops
// at the first positional, which would silently drop flags like
// `jira update PROJ-1 --priority Major`. It returns the positionals.
func parseFlags(fs *flag.FlagSet, args []string) []string {
	isBool := func(name string) bool {
		f := fs.Lookup(name)
		if f == nil {
			return false
		}
		if bv, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
			return bv.IsBoolFlag()
		}
		return false
	}

	var positional, reordered []string
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if isOption(a) {
			name := strings.TrimPrefix(strings.TrimPrefix(a, "--"), "-")
			if strings.Contains(name, "=") {
				reordered = append(reordered, a)
				i++
				continue
			}
			reordered = append(reordered, a)
			i++
			if !isBool(name) && i < len(args) && !isOption(args[i]) {
				reordered = append(reordered, args[i])
				i++
			}
			continue
		}
		positional = append(positional, a)
		i++
	}
	fs.Parse(reordered)
	return positional
}

// isOption reports whether a looks like a flag rather than a value. A bare
// "-" and negative numbers are values, as in argparse.
func isOption(a string) bool {
	if !strings.HasPrefix(a, "-") || a == "-" {
		return false
	}
	return !negativeNumber.MatchString(a)
}
