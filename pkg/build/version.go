package build

import "fmt"

// This gets populated through build flags
var Build string

type VersionDescriptor struct {
	Major int
	Minor int
	Patch int
}

func (o *VersionDescriptor) String() string {
	ret := fmt.Sprintf("%d.%d.%d", o.Major, o.Minor, o.Patch)

	if Build != "" {
		ret += fmt.Sprintf(" (%s)", Build)
	}

	return ret
}

var Version = VersionDescriptor{
	Major: 0,
	Minor: 0,
	Patch: 1,
}
