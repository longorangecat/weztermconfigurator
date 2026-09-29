package catalog

import "testing"

// Every choice of every Enum/Flags field (options, nested schemas, action
// arguments) must be explained, since the UI shows a "?" for each one.
func TestEveryChoiceHasDoc(t *testing.T) {
	var walk func(path string, f *Field)
	walk = func(path string, f *Field) {
		if f.Kind == Enum || f.Kind == Flags {
			for _, v := range f.Enum {
				if ValueDoc(f.Name, v) == "" {
					t.Errorf("%s: no doc for %s/%s", path, f.Name, v)
				}
			}
		}
		for i := range f.Fields {
			walk(path+"."+f.Fields[i].Name, &f.Fields[i])
		}
	}
	for i := range Options {
		walk(Options[i].Name, &Options[i].Field)
	}
	for i := range Actions {
		if Actions[i].Arg != nil {
			walk("action "+Actions[i].Name, Actions[i].Arg)
		}
	}
}
