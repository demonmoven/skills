package repoinfo

import "regexp"

var UnusedImportRegexp = []*regexp.Regexp{
	regexp.MustCompile("^(.*) imported but not used$"),
	regexp.MustCompile(`^(.*) imported but not used as (\w+)$`),
	regexp.MustCompile("^(.*) imported and not used$"),          // Go 1.20+
	regexp.MustCompile(`^(.*) imported as (\w+) and not used$`), // Go 1.20+
}

var UnusedVariableRegexp = []*regexp.Regexp{
	regexp.MustCompile("^(.*) declared but not used$"),
	regexp.MustCompile("^(.*) declared and not used$"),  // Go 1.20+
	regexp.MustCompile("^declared and not used: (.*)$"), // Go 1.23+
}

var RedeclaredMains = []*regexp.Regexp{
	regexp.MustCompile("^main redeclared in this block"),
	regexp.MustCompile("^\tother declaration of main"),
}
