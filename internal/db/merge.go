package db

// Merge combines an incremental collection with the previous database.
//
// fresh holds the current snapshot and the tests that were just re-collected.
// Every other test in keep takes its record from old: it was not selected, so
// nothing it depends on changed and its dependencies are still valid for the
// current snapshot. Tests in neither (deleted) are dropped.
func Merge(old, fresh *DB, keep map[string]bool) *DB {
	out := *fresh
	out.Tests = map[string]*Test{}
	out.Funcs = append([]string(nil), fresh.Funcs...)
	idx := make(map[string]int, len(out.Funcs))
	for i, f := range out.Funcs {
		idx[f] = i
	}
	for id, t := range fresh.Tests {
		out.Tests[id] = t
	}
	for id := range keep {
		if _, ok := out.Tests[id]; ok {
			continue
		}
		t, ok := old.Tests[id]
		if !ok {
			continue
		}
		cp := *t
		cp.Funcs = make([]int, 0, len(t.Funcs))
		for _, fi := range t.Funcs {
			if fi >= len(old.Funcs) {
				continue
			}
			name := old.Funcs[fi]
			i, ok := idx[name]
			if !ok {
				i = len(out.Funcs)
				idx[name] = i
				out.Funcs = append(out.Funcs, name)
			}
			cp.Funcs = append(cp.Funcs, i)
		}
		out.Tests[id] = &cp
	}
	return &out
}
