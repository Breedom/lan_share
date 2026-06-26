package core

import (
	"fmt"
	"os/exec"
	"syscall"
)

func EnsureFirewallRule(port int) {
	ruleName := "LanShare"

	checkCmd := exec.Command("netsh", "advfirewall", "firewall", "show", "rule", "name="+ruleName)
	checkCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := checkCmd.Run(); err == nil {
		fmt.Println("[✓] 防火墙规则已存在")
		return
	}

	fmt.Printf("[*] 首次运行，正在添加防火墙规则 (端口 %d)...\n", port)

	addFirewallRule(ruleName, port)
}

func addFirewallRule(name string, port int) {
	rules := []string{
		fmt.Sprintf("netsh advfirewall firewall add rule name=\"%s\" dir=in action=allow protocol=TCP localport=%d", name, port),
		fmt.Sprintf("netsh advfirewall firewall add rule name=\"%s (UDP)\" dir=in action=allow protocol=UDP localport=%d", name, port),
	}

	for _, rule := range rules {
		cmd := exec.Command("cmd", "/c", rule)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Run(); err != nil {
			fmt.Printf("[!] 防火墙规则添加失败，请以管理员身份运行: %v\n", err)
			fmt.Println("[*] 或手动运行: netsh advfirewall firewall add rule name=\"LanShare\" dir=in action=allow protocol=TCP localport=8080")
			return
		}
	}

	fmt.Printf("[✓] 防火墙规则已添加 (端口 %d)\n", port)
}
