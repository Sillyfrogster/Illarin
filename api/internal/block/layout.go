package block

type Layout string

const (
	Single    Layout = "single"
	Duo       Layout = "duo"
	MainAside Layout = "main-aside"
	Trio      Layout = "trio"
	Stack2    Layout = "stack-2"
	Stack3    Layout = "stack-3"
)

var slots = map[Layout][]Slot{
	Single:    {"main"},
	Duo:       {"left", "right"},
	MainAside: {"main", "aside"},
	Trio:      {"left", "middle", "right"},
	Stack2:    {"top", "bottom"},
	Stack3:    {"top", "middle", "bottom"},
}

var minimumWidths = map[Layout]Width{
	Single:    Third,
	Duo:       TwoThirds,
	MainAside: TwoThirds,
	Trio:      Full,
	Stack2:    Third,
	Stack3:    Third,
}

func (l Layout) Slots() []Slot { return slots[l] }

func (l Layout) MinimumWidth() Width { return minimumWidths[l] }

type Width string

const (
	Full      Width = "full"
	TwoThirds Width = "two_thirds"
	Half      Width = "half"
	Third     Width = "third"
)

var widthColumns = map[Width]int{
	Full:      12,
	TwoThirds: 8,
	Half:      6,
	Third:     4,
}

func (w Width) Columns() int { return widthColumns[w] }

func (w Width) label() string {
	switch w {
	case Full:
		return "full width"
	case TwoThirds:
		return "two thirds"
	case Half:
		return "half"
	case Third:
		return "a third"
	default:
		return string(w)
	}
}
