// Package netutil 提供起服前的网络辅助检查。
package netutil

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CheckPortOccupied 探测 addr（host:port）是否已被别的进程监听。
//
// 为什么用「主动拨号」而不是看 net.Listen 是否报错：Windows 下 Go 的 net.Listen 默认带
// SO_REUSEADDR，端口已被占用时它仍会「成功」返回、造成两个进程静默双绑同一端口，新进来的连接
// 被先前的监听者抢走——正是本项目遇到过的 UnrealEditor 抢占 8888 导致 GlobalServer 注册全超时的坑。
// 主动拨号：能连上 = 已有监听者（被占用）；连接被拒 = 端口空闲。该判定不受 SO_REUSEADDR 影响。
//
// 返回是否被占用，以及占用方描述（Windows 尽力用 netstat+tasklist 查出 PID/进程名；查不到则为空）。
func CheckPortOccupied(addr string) (occupied bool, occupant string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false, ""
	}
	// 0.0.0.0 / :: / 空 表示监听所有网卡，拨号走本地回环。
	dialHost := host
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		dialHost = "127.0.0.1"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(dialHost, port), 500*time.Millisecond)
	if err != nil {
		return false, "" // 连不上 = 端口空闲
	}
	_ = conn.Close()
	return true, describeOccupant(port)
}

// EnsurePortFree 起服前调用：端口被占用则返回带占用方信息的 error（供调用方记日志并 fail-fast）。
func EnsurePortFree(addr string) error {
	occupied, occupant := CheckPortOccupied(addr)
	if !occupied {
		return nil
	}
	if occupant != "" {
		return fmt.Errorf("端口 %s 已被占用（占用方 %s）；拒绝启动以避免 SO_REUSEADDR 静默双绑、连接被劫持", addr, occupant)
	}
	return fmt.Errorf("端口 %s 已被占用（占用方进程未能识别）；拒绝启动以避免 SO_REUSEADDR 静默双绑、连接被劫持", addr)
}

// describeOccupant 尽力查出监听 port 的进程（PID + 进程名）。仅 Windows；其它平台或查不到返回空。
func describeOccupant(port string) string {
	if runtime.GOOS != "windows" {
		return ""
	}
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return ""
	}
	pid := ""
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		// 列：Proto LocalAddr ForeignAddr State PID
		if len(f) >= 5 && f[0] == "TCP" && strings.HasSuffix(f[1], ":"+port) && f[3] == "LISTENING" {
			pid = f[4]
			break
		}
	}
	if pid == "" {
		return ""
	}
	name := processName(pid)
	if name != "" {
		return fmt.Sprintf("PID=%s (%s)", pid, name)
	}
	return "PID=" + pid
}

// processName 用 tasklist 查 PID 对应的进程名（Windows）。
func processName(pid string) string {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+pid, "/FO", "CSV", "/NH").Output()
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(out))
	if s == "" || strings.HasPrefix(s, "INFO:") {
		return "" // 未找到该 PID
	}
	// CSV: "映像名称","PID",...
	parts := strings.SplitN(s, ",", 2)
	if len(parts) == 0 {
		return ""
	}
	return strings.Trim(parts[0], "\"")
}
