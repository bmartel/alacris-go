package main

import "runtime/debug"

// version reports the module version this binary was built from, which is what
// a user needs when a generated file looks wrong.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			if len(s.Value) > 12 {
				return "(devel " + s.Value[:12] + ")"
			}
			return "(devel " + s.Value + ")"
		}
	}
	return "(devel)"
}
