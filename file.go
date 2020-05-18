package main

type File struct {
	Filename      string
	Organizations []Organization `yaml:"organizations"`
	Repositories  []Repository   `yaml:"repositories"`
	Bases         []Base         `yaml:"bases"`
}

type Base struct {
	Name       string     `yaml:"name"`
	Repository Repository `yaml:"repository"`
}
