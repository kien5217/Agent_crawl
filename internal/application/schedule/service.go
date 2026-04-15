package schedule

import "context"

type Discoverer interface {
	Name() string
	Enqueue(ctx context.Context) (int, error)
}

type Result struct {
	Counts map[string]int
}

type DiscovererError struct {
	Discoverer string
	Err        error
}

func (e *DiscovererError) Error() string {
	return e.Discoverer + ": " + e.Err.Error()
}

func (e *DiscovererError) Unwrap() error {
	return e.Err
}

type Service struct {
	discoverers []Discoverer
}

func NewService(discoverers ...Discoverer) *Service {
	return &Service{discoverers: discoverers}
}

func (s *Service) Run(ctx context.Context) (Result, error) {
	result := Result{Counts: make(map[string]int, len(s.discoverers))}
	for _, discoverer := range s.discoverers {
		count, err := discoverer.Enqueue(ctx)
		if err != nil {
			return result, &DiscovererError{Discoverer: discoverer.Name(), Err: err}
		}
		result.Counts[discoverer.Name()] = count
	}
	return result, nil
}
