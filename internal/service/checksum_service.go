package service

type ChecksumService struct{}

func NewChecksumService() *ChecksumService {
	return &ChecksumService{}
}

func (s *ChecksumService) Checksum() (string, error) {
	return "", nil
}
