package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var (
	v           bool
	currentUser string
	ctx         = context.Background()
	client      *github.Client
	bases       []base
)

type Sync struct {
	files         []string
	verbose       bool
	token         string
	confirmPublic bool
}

// Exec function parse and process yaml file(s)
func (s *Sync) Exec() {
	v = s.verbose
	client = newGithubClient(s.token)
	var parsedFiles []*file

	u, _, err := client.Users.Get(ctx, "")
	if err != nil {
		log.Fatalf("Unable to retrieve current github user, error: %s", err)
	}
	currentUser = *u.Login

	for _, arg := range s.files {
		f := new(file)
		f.Filename = path.Base(arg)
		parsedFiles = append(parsedFiles, f.getFile(arg))
	}

	for _, obj := range parsedFiles {
		fmt.Println("Processing file", obj.Filename)

		for _, repo := range obj.Repositories {
			logIfVerbose(fmt.Sprintf("Sync repo %s...", repo.Name))
			appendBaseToRepo(&repo, parsedFiles)
			processRepo(repo, "", s.confirmPublic)
		}

		for _, org := range obj.Organizations {
			logIfVerbose(fmt.Sprintf("Sync organization %s...", org.Name))
			syncOrgTeams(org)

			for _, orgRepo := range org.Repositories {
				logIfVerbose(fmt.Sprintf("Sync repo %s on organization %s...", orgRepo.Name, org.Name))
				appendBaseToRepo(&orgRepo, parsedFiles)
				processRepo(orgRepo, org.Name, s.confirmPublic)
			}
		}

	}
}

func createRepo(org string, t *github.Repository) (*github.Repository, *github.Response, error) {
	return client.Repositories.Create(ctx, org, t)
}

func editRepo(org string, t *github.Repository) (*github.Repository, *github.Response, error) {
	if org == "" {
		org = currentUser
	}
	return client.Repositories.Edit(ctx, org, *t.Name, t)
}

func editRepoBranches(org string, repo string, branch string, t *github.ProtectionRequest) (*github.Protection, *github.Response, error) {
	if org == "" {
		org = currentUser
	}
	return client.Repositories.UpdateBranchProtection(ctx, org, repo, branch, t)
}

func (f *file) getFile(filePath string) *file {
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &f)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return f
}

func newGithubClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
}

func logIfVerbose(message string) {
	if v {
		log.Println(message)
	}
}
