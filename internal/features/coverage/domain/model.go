package domain

type Rule struct {
	Pattern string  `json:"pattern"`
	Min     float64 `json:"min"`
}

type Package struct {
	ImportPath string
	Dir        string
	Files      []string
	FirstFile  string
}

type Block struct {
	File       string
	Position   string
	Statements int64
	Covered    bool
}

type Coverage struct {
	Profile []byte
	Blocks  []Block
}

type Result struct {
	ImportPath string
	File       string
	Rule       *Rule
	Coverage   float64
	Statements int64
	Covered    int64
	Skipped    bool
	Violation  bool
	Message    string
}

type Report struct {
	Results []Result
	Checked int
	Failed  int
	Skipped int
}
