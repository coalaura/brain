package brain

type Index struct {
	index   int
	indices map[string]int
}

func NewIndex(max int) *Index {
	return &Index{
		index:   max,
		indices: make(map[string]int),
	}
}

func (i *Index) SetById(id string) int {
	idx := i.index
	i.index--

	i.indices[id] = idx

	return idx
}

func (i *Index) FindById(id string) (int, bool) {
	idx, ok := i.indices[id]

	return idx, ok
}
