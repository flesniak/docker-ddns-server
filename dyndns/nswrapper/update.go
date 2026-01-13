package nswrapper

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/benjaminbear/docker-ddns-server/dyndns/model"
	l "github.com/labstack/gommon/log"
)

func UpdateHost(host model.Host, enableWildcard bool) {
	if host.Ip4 != "" {
		if err := UpdateRecord(host.Hostname, host.Ip4, "A", host.Domain, host.Ttl, enableWildcard); err != nil {
			l.Error(fmt.Sprintf("DNS error: %v", err))
		}
	}

	if host.Ip6 != "" {
		if err := UpdateRecord(host.Hostname, host.Ip6, "AAAA", host.Domain, host.Ttl, enableWildcard); err != nil {
			l.Error(fmt.Sprintf("DNS error: %v", err))
		}
	}
}

// UpdateRecord builds a nsupdate file and updates a record by executing it with nsupdate.
func UpdateRecord(hostname string, target string, addrType string, zone string, ttl int, enableWildcard bool) error {
	l.Info(fmt.Sprintf("%s record update request: %s -> %s", addrType, hostname, target))

	f, err := os.CreateTemp(os.TempDir(), "dyndns")
	if err != nil {
		return err
	}

	defer os.Remove(f.Name())
	w := bufio.NewWriter(f)

	w.WriteString(fmt.Sprintf("server %s\n", "localhost"))
	w.WriteString(fmt.Sprintf("zone %s\n", zone))
	w.WriteString(fmt.Sprintf("update delete %s.%s %s\n", hostname, zone, addrType))
	if enableWildcard {
		w.WriteString(fmt.Sprintf("update delete %s.%s %s\n", "*."+hostname, zone, addrType))
	}
	w.WriteString(fmt.Sprintf("update add %s.%s %v %s %s\n", hostname, zone, ttl, addrType, target))
	if enableWildcard {
		w.WriteString(fmt.Sprintf("update add %s.%s %v %s %s\n", "*."+hostname, zone, ttl, addrType, target))
	}
	w.WriteString("send\n")

	w.Flush()
	f.Close()

	cmd := exec.Command("/usr/bin/nsupdate", f.Name())
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %v", err, stderr.String())
	}

	if out.String() != "" {
		return errors.New(out.String())
	}

	return nil
}

// DeleteRecord builds a nsupdate file and deletes a record by executing it with nsupdate.
func DeleteRecord(hostname string, zone string, enableWildcard bool) error {
	fmt.Printf("record delete request: %s\n", hostname)

	f, err := os.CreateTemp(os.TempDir(), "dyndns")
	if err != nil {
		return err
	}

	defer os.Remove(f.Name())
	w := bufio.NewWriter(f)

	w.WriteString(fmt.Sprintf("server %s\n", "localhost"))
	w.WriteString(fmt.Sprintf("zone %s\n", zone))
	w.WriteString(fmt.Sprintf("update delete %s.%s\n", hostname, zone))
	if enableWildcard {
		w.WriteString(fmt.Sprintf("update delete %s.%s\n", "*."+hostname, zone))
	}
	w.WriteString("send\n")

	w.Flush()
	f.Close()

	cmd := exec.Command("/usr/bin/nsupdate", f.Name())
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %v", err, stderr.String())
	}

	if out.String() != "" {
		return errors.New(out.String())
	}

	return nil
}
