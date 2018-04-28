package saltboot

import (
	"log"
	"strings"
)

const (
	UBUNTU        = "Ubuntu"
	DEBIAN        = "Debian"
	SUSE          = "SUSE"
	SLES12        = "sles12"
	AMAZONLINUX_2 = "amazonlinux2"
	AZURE         = "AZURE"
)

type closable interface {
	Close() error
}

func closeIt(target closable) {
	if err := target.Close(); err != nil {
		log.Printf("[Utils] [ERROR] couldn't close target: %s", err.Error())
	}
}

func isOs(os *Os, name ...string) bool {
	match := false
	for _, n := range name {
		if os == nil || (os != nil && len(os.Name) == 0) {
			log.Printf("[isOs] find out if os is %s", n)
			match = isOsMatch(n)
		} else {
			log.Printf("[isOs] match if os is %s", n)
			match = containsLowerCase(os.Name, n)
		}
		if match {
			log.Printf("[isOs] os is %s", n)
			break
		} else {
			log.Printf("[isOs] os is not %s", n)
		}
	}
	return match
}

// Deprecated: Do not use this function, OS type is expected to be provided
func isOsMatch(os string) bool {
	out, _ := ExecCmd("grep", os, "/etc/issue")
	if len(out) > 0 {
		log.Printf("[isOsMatch] host OS is determined to be %s", os)
		return true
	}
	return false
}

func containsLowerCase(name, substr string) bool {
	return strings.Contains(strings.ToLower(name), strings.ToLower(substr))
}

func isCloud(name string, cloud *Cloud) bool {
	return cloud != nil && strings.ToLower(cloud.Name) == strings.ToLower(name)
}
