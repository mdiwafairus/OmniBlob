package service

type TransferService struct{}

func NewTransferService() *TransferService {
	return &TransferService{}
}

func (s *TransferService) Transfer() error {
	return nil
}
