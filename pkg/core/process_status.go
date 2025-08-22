package core

import (
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/core/server"
)

func StartProcess(cache *server.Cache, processKey string) error {
	return cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING, "")
}

func RunningProcess(cache *server.Cache, processKey string) error {
	return cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_RUNNING, "")
}

func SleepingProcess(cache *server.Cache, processKey string) error {
	return cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_SLEEPING, "")
}

func CompleteProcess(cache *server.Cache, processKey string) error {
	return cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_COMPLETED, "")
}

func ErrorProcess(cache *server.Cache, processKey string, errorMsg string) error {
	return cache.UpdateProcessState(processKey, v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_ERROR, errorMsg)
}