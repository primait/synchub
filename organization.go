package main

type organization struct {
	Name         string `yaml:"name"`
	Repositories []repository
}
