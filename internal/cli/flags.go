package cli

type strPtr struct{ p **string }

func (f *strPtr) String() string {
	if f.p != nil && *f.p != nil {
		return **f.p
	}
	return ""
}
func (f *strPtr) Set(s string) error { v := s; *f.p = &v; return nil }

type repoFlag struct{ p *[]string }

func (f *repoFlag) String() string     { return "" }
func (f *repoFlag) Set(s string) error { *f.p = append(*f.p, s); return nil }
