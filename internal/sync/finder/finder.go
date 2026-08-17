package finder

type Finder struct{}

func NewFinder() *Finder {
	return &Finder{}
}

func (f *Finder) Find() ([]string, error) {
	return nil, nil
}
