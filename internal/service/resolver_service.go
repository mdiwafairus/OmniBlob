package service

type ResolverService struct{}

func NewResolverService() *ResolverService {
	return &ResolverService{}
}

func (s *ResolverService) Resolve() error {
	return nil
}
