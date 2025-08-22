package server

import (
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
)

func (s *Server) StartProcess(processKey string) error {
	return s.cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING, "")
}

func (s *Server) RunningProcess(processKey string) error {
	return s.cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_RUNNING, "")
}

func (s *Server) SleepingProcess(processKey string) error {
	return s.cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_SLEEPING, "")
}

func (s *Server) CompleteProcess(processKey string) error {
	return s.cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_COMPLETED, "")
}

func (s *Server) ErrorProcess(processKey string, errorMsg string) error {
	return s.cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_ERROR, errorMsg)
}
