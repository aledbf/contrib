/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"text/template"

	"github.com/golang/glog"
	k8sexec "k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/iptables"
)

const (
	iptablesChain = "KUBE-KEEPALIVED-VIP"
	keepalivedCfg = "/etc/keepalived/keepalived.conf"
	haproxyCfg    = "/etc/haproxy/haproxy.cfg"
)

var (
	keepalivedTmpl = "keepalived.tmpl"
	haproxyTmpl    = "haproxy.tmpl"
)

type keepalived struct {
	ip             string
	iface          string
	netmask        int
	priority       int
	nodes          []string
	neighbors      []string
	vips           []string
	useUnicast     bool
	started        bool
	proxyMode      bool
	keepalivedTmpl *template.Template
	haproxyTmpl    *template.Template
	cmd            *exec.Cmd
	ipt            iptables.Interface
}

// WriteCfg creates a new keepalived configuration file.
// In case of an error with the generation it returns the error
func (k *keepalived) WriteCfg(svcs []vip) error {
	w, err := os.Create(keepalivedCfg)
	if err != nil {
		return err
	}

	k.vips = getVIPs(svcs)

	conf := make(map[string]interface{})
	conf["iptablesChain"] = iptablesChain
	conf["iface"] = k.iface
	conf["myIP"] = k.ip
	conf["netmask"] = k.netmask
	conf["svcs"] = svcs
	conf["vips"] = getVIPs(svcs)
	conf["nodes"] = k.neighbors
	conf["priority"] = k.priority
	conf["useUnicast"] = k.useUnicast
	conf["proxyMode"] = k.proxyMode

	if glog.V(2) {
		b, _ := json.Marshal(conf)
		glog.Infof("%v", string(b))
	}

	err = k.keepalivedTmpl.Execute(w, conf)
	w.Close()
	if err != nil {
		return fmt.Errorf("unexpected error creating keepalived.cfg: %v", err)
	}

	if k.proxyMode {
		w, err := os.Create(haproxyCfg)
		if err != nil {
			return err
		}

		err = k.haproxyTmpl.Execute(w, conf)
		w.Close()
		if err != nil {
			return fmt.Errorf("unexpected error creating haproxy.cfg: %v", err)
		}
	}

	return nil
}

// getVIPs returns a list of the virtual IP addresses to be used in keepalived
// without duplicates (a service can use more than one port)
func getVIPs(svcs []vip) []string {
	result := []string{}
	for _, svc := range svcs {
		result = appendIfMissing(result, svc.IP)
	}

	return result
}

// Start starts a keepalived process in foreground.
// In case of any error it will terminate the execution with a fatal error
func (k *keepalived) Start() {
	ae, err := k.ipt.EnsureChain(iptables.TableFilter, iptables.Chain(iptablesChain))
	if err != nil {
		glog.Fatalf("unexpected error: %v", err)
	}
	if ae {
		glog.V(2).Infof("chain %v already existed", iptablesChain)
	}

	k.cmd = exec.Command("/usr/sbin/keepalived",
		"--dont-fork",
		"--log-console",
		"--release-vips",
		"--pid", "/keepalived.pid")

	k.cmd.Stdout = os.Stdout
	k.cmd.Stderr = os.Stderr

	if err := k.cmd.Start(); err != nil {
		glog.Errorf("keepalived error: %v", err)
	}

	k.started = true

	if err := k.cmd.Wait(); err != nil {
		glog.Fatalf("keepalived error: %v", err)
	}
}

// Reload sends SIGHUP to keepalived to reload the configuration.
func (k *keepalived) Reload() error {
	if !k.started {
		// TODO: add a warning indicating that keepalived is not started?
		return nil
	}

	glog.Info("reloading keepalived")
	err := syscall.Kill(k.cmd.Process.Pid, syscall.SIGHUP)
	if err != nil {
		return fmt.Errorf("error reloading keepalived: %v", err)
	}

	return nil
}

// Stop stop keepalived process
func (k *keepalived) Stop() {
	for _, vip := range k.vips {
		k.removeVIP(vip)
	}

	err := k.ipt.FlushChain(iptables.TableFilter, iptables.Chain(iptablesChain))
	if err != nil {
		glog.V(2).Infof("unexpected error flushing iptables chain %v: %v", err, iptablesChain)
	}

	err = syscall.Kill(k.cmd.Process.Pid, syscall.SIGTERM)
	if err != nil {
		glog.Errorf("error stopping keepalived: %v", err)
	}
}

func resetIPVS() error {
	glog.Info("cleaning ipvs configuration")
	_, err := k8sexec.New().Command("ipvsadm", "-C").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error removing ipvs configuration: %v", err)
	}

	return nil
}

func (k *keepalived) removeVIP(vip string) error {
	glog.Infof("removing configured VIP %v", vip)
	out, err := k8sexec.New().Command("ip", "addr", "del", vip+"/32", "dev", k.iface).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error reloading keepalived: %v\n%s", err, out)
	}
	return nil
}

func (k *keepalived) loadTemplates() error {
	tmpl, err := template.ParseFiles(keepalivedTmpl)
	if err != nil {
		return err
	}
	k.keepalivedTmpl = tmpl

	tmpl, err = template.ParseFiles(haproxyTmpl)
	if err != nil {
		return err
	}
	k.haproxyTmpl = tmpl

	return nil
}
