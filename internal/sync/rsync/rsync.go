package rsync

type Rsync struct{}

func NewRsync() *Rsync {
	return &Rsync{}
}

func (r *Rsync) Sync(source, destination string) error {
	return nil
}
