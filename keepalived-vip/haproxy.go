package main

type haproxy struct {
	ReloadCmd string `json:"reloadCmd" description:"command used to reload the load balancer."`
	Template  string `json:"template" description:"template for the load balancer config."`
}

// reload reloads the loadbalancer using the reload cmd specified in the json manifest.
func (cfg *haproxy) reload() error {
	output, err := exec.Command("sh", "-c", cfg.ReloadCmd).CombinedOutput()
	msg := fmt.Sprintf("%v -- %v", cfg.Name, string(output))
	if err != nil {
		return fmt.Errorf("error restarting %v: %v", msg, err)
	}

	return nil
}

// sync all services with the loadbalancer.
func (lbc *loadBalancerController) sync(tmpl) error {
	if err := lbc.cfg.write(
		map[string][]service{
			"http":      httpSvc,
			"httpsTerm": httpsTermSvc,
			"tcp":       tcpSvc,
		}, dryRun); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	return lbc.cfg.reload()
}

// write writes the configuration file, will write to stdout if dryRun == true
func (cfg *loadBalancerConfig) write(services map[string][]service, dryRun bool) (err error) {
	var w io.Writer
	if dryRun {
		w = os.Stdout
	} else {
		w, err = os.Create(cfg.Config)
		if err != nil {
			return
		}
	}
	var t *template.Template
	t, err = template.ParseFiles(cfg.Template)
	if err != nil {
		return
	}

	conf := make(map[string]interface{})
	conf["services"] = services
	err = t.Execute(w, conf)
	return
}
