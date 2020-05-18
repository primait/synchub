package main

type Organization struct {
	Name         string `yaml:"name"`
	Repositories []Repository
}
