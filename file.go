package main

type file struct {
	Filename      string
	Organizations []organization `yaml:"organizations"`
	Repositories  []repository   `yaml:"repositories"`
	Bases         []base         `yaml:"bases"`
}

type base struct {
	Name       string     `yaml:"name"`
	Repository repository `yaml:"repository"`
}
