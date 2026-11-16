package service

// RequestRestart asks the process that owns the service to shut down through
// its normal path and start again, for example after an update was applied.
// It is safe to call more than once.
func (s *Service) RequestRestart() {
	s.restartOnce.Do(func() { close(s.restart) })
}

// RestartRequested is closed once RequestRestart was called. The process
// owner waits on it next to its own stop signals.
func (s *Service) RestartRequested() <-chan struct{} {
	return s.restart
}
