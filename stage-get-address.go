package main

import (
	"flag"
	"fmt"
	"time"
)

type vaiGetAddress struct {
	Timeout time.Duration
}

func newGetAddressStage() (Stage, error) {
	return &vaiGetAddress{}, nil
}

func (*vaiGetAddress) Name() string {
	return "get-address"
}

func (ga *vaiGetAddress) Configure(flags *flag.FlagSet) {
	flags.DurationVar(&ga.Timeout, "t", 0, "timeout")
}

func (ga *vaiGetAddress) Validate() error {
	switch {
	case ga.Timeout < 0:
		return fmt.Errorf("timeout (%v) may not be < 0", ga.Timeout)
	}
	return nil
}

func (ga *vaiGetAddress) Run() error {
	const (
		hookTarget = "/usr/libexec/dhcpcd-hooks/20-resolv.conf"
		hookDest   = "/usr/lib/dhcpcd-hooks/20-resolv.conf"
	)
	if err := osMkdirAll("/usr/lib/dhcpcd/dhcpcd-hooks", 0755); err != nil {
		return err
	}
	if err := symlink(hookTarget, hookDest, true); err != nil {
		return err
	}
	if err := runCmd("dhcpcd", "-w", "-l", "--timeout", Seconds(ga.Timeout)); err != nil {
		return err
	}
	return nil
}
