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
	bases       []Base
)

func Sync(files []string, verbose bool, token string) {
	v = verbose
	client = newGithubClient(token)
	var parsedFiles []*File

	u, _, err := client.Users.Get(ctx, "")
	if err != nil {
		log.Fatalf("Unable to retrieve current github user, error: %s", err)
	}
	currentUser = *u.Login

	for _, arg := range files {
		f := new(File)
		f.Filename = path.Base(arg)
		parsedFiles = append(parsedFiles, f.getFile(arg))
	}

	for _, obj := range parsedFiles {
		fmt.Println("Processing file", obj.Filename)

		for _, repo := range obj.Repositories {
			logIfVerbose(fmt.Sprintf("Sync repo %s...", repo.Name))
			AppendBaseToRepo(&repo, parsedFiles)
			ProcessRepo(repo, "")
		}

		for _, org := range obj.Organizations {
			logIfVerbose(fmt.Sprintf("Sync organization %s...", org.Name))
			for _, orgRepo := range org.Repositories {
				logIfVerbose(fmt.Sprintf("Sync repo %s on organization %s...", orgRepo.Name, org.Name))
				AppendBaseToRepo(&orgRepo, parsedFiles)
				ProcessRepo(orgRepo, org.Name)
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

func (f *File) getFile(filePath string) *File {
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
