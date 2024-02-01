//go:build linux

package process

import (
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/agent/component/discovery"
	"github.com/grafana/agent/component/discovery/process/analyze"
	analCache "github.com/grafana/agent/component/discovery/process/analyze/cache"
	gopsutil "github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/unix"
)

const (
	labelProcessID          = "__process_pid__"
	labelProcessExe         = "__meta_process_exe"
	labelProcessCwd         = "__meta_process_cwd"
	labelProcessCommandline = "__meta_process_commandline"
	labelProcessUsername    = "__meta_process_username"
	labelProcessUID         = "__meta_process_uid"
	labelProcessContainerID = "__container_id__"
)

type Process struct {
	PID         string
	exe         string
	cwd         string
	commandline string
	containerID string
	username    string
	uid         string
	Analysis    *analyze.Results
}

func (p Process) String() string {
	return fmt.Sprintf("pid=%s exe=%s cwd=%s commandline=%s containerID=%s", p.PID, p.exe, p.cwd, p.commandline, p.containerID)
}

func convertProcesses(ps []Process) []discovery.Target {
	var res []discovery.Target
	for _, p := range ps {
		t := convertProcess(p)
		res = append(res, t)
	}
	return res
}

func convertProcess(p Process) discovery.Target {
	t := make(discovery.Target, 5)
	t[labelProcessID] = p.PID
	if p.exe != "" {
		t[labelProcessExe] = p.exe
	}
	if p.cwd != "" {
		t[labelProcessCwd] = p.cwd
	}
	if p.commandline != "" {
		t[labelProcessCommandline] = p.commandline
	}
	if p.containerID != "" {
		t[labelProcessContainerID] = p.containerID
	}
	if p.username != "" {
		t[labelProcessUsername] = p.username
	}
	if p.uid != "" {
		t[labelProcessUID] = p.uid
	}
	for k, v := range p.Analysis.Labels {
		t[k] = v
	}

	return t
}

func Discover(l log.Logger, cfg *DiscoverConfig, cache *analCache.Cache) ([]Process, error) {
	processes, err := gopsutil.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}
	res := make([]Process, 0, len(processes))
	loge := func(pid int, e error) {
		if errors.Is(e, unix.ESRCH) {
			return
		}
		if errors.Is(e, os.ErrNotExist) {
			return
		}
		_ = level.Error(l).Log("msg", "failed to get process info", "err", e, "pid", pid)
	}
	active := make(map[uint32]struct{})
	for _, p := range processes {
		spid := fmt.Sprintf("%d", p.Pid)
		var (
			exe, cwd, commandline, containerID, username, uid string
		)
		if cfg.Exe {
			exe, err = p.Exe()
			if err != nil {
				loge(int(p.Pid), err)
				continue
			}
		}
		if cfg.Cwd {
			cwd, err = p.Cwd()
			if err != nil {
				loge(int(p.Pid), err)
				continue
			}
		}
		if cfg.Commandline {
			commandline, err = p.Cmdline()
			if err != nil {
				loge(int(p.Pid), err)
				continue
			}
		}
		if cfg.Username {
			username, err = p.Username()
			if err != nil {
				loge(int(p.Pid), err)
				continue
			}
		}
		if cfg.UID {
			uids, err := p.Uids()
			if err != nil {
				loge(int(p.Pid), err)
				continue
			}
			if len(uids) > 0 {
				uid = fmt.Sprintf("%d", uids[0])
			}
		}

		if cfg.ContainerID {
			containerID, err = getLinuxProcessContainerID(spid)
			if err != nil {
				loge(int(p.Pid), err)
				continue
			}
		}
		m, err := cache.AnalyzePID(spid)
		if err != nil {
			level.Error(l).Log("msg", "error analyzing process", "pid", spid, "err", err)
			continue
		}
		res = append(res, Process{
			PID:         spid,
			exe:         exe,
			cwd:         cwd,
			commandline: commandline,
			containerID: containerID,
			username:    username,
			uid:         uid,
			Analysis:    m,
		})
		active[uint32(p.Pid)] = struct{}{}
	}
	cache.GC(active)

	return res, nil
}

func getLinuxProcessContainerID(pid string) (string, error) {
	if runtime.GOOS == "linux" {
		cgroup, err := os.Open(path.Join("/proc", pid, "cgroup"))
		if err != nil {
			return "", err
		}
		defer cgroup.Close()
		cid := getContainerIDFromCGroup(cgroup)
		if cid != "" {
			return cid, nil
		}
	}
	return "", nil
}
