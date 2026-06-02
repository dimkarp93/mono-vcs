package cli

import "strconv"

// These flag.Value types set their pointer only when the flag is present, so an
// absent flag stays nil and config defaults can fill it later.

type strPtr struct{ p **string }

func (f *strPtr) String() string {
	if f.p != nil && *f.p != nil {
		return **f.p
	}
	return ""
}
func (f *strPtr) Set(s string) error { v := s; *f.p = &v; return nil }

type intPtr struct{ p **int }

func (f *intPtr) String() string {
	if f.p != nil && *f.p != nil {
		return strconv.Itoa(**f.p)
	}
	return ""
}
func (f *intPtr) Set(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*f.p = &n
	return nil
}

type boolPtr struct{ p **bool }

func (f *boolPtr) String() string {
	if f.p != nil && *f.p != nil {
		return strconv.FormatBool(**f.p)
	}
	return "false"
}
func (f *boolPtr) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*f.p = &b
	return nil
}
func (f *boolPtr) IsBoolFlag() bool { return true }

type repoFlag struct{ p *[]string }

func (f *repoFlag) String() string     { return "" }
func (f *repoFlag) Set(s string) error { *f.p = append(*f.p, s); return nil }
