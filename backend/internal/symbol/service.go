package symbol

type Service struct {
	provider Provider
}

func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) Search(query, exchange string, page, pageSize int) ([]Symbol, int) {
	all := s.provider.Search(query, exchange)
	total := len(all)
	start := (page - 1) * pageSize
	if start >= total {
		return []Symbol{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total
}

func (s *Service) Detail(sym string) (Detail, bool) {
	return s.provider.Detail(sym)
}
