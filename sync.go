package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var (
	v           bool
	isCI        bool
	currentUser string
	ctx         = context.Background()
	client      *github.Client
	bases       []base
)

type Sync struct {
	files         []string
	token         string
	repos         string
	verbose       bool
	confirmPublic bool
	skipPublic    bool
	isCI          bool
}

// Exec function parse and process yaml file(s)
func (s *Sync) Exec() {
	v = s.verbose
	isCI = s.isCI
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

	restrictedRepos := strings.Split(s.repos, ",")

	sp.FinalMSG = "✔️ Synchronization completed!"

	for _, obj := range parsedFiles {
		sp.Suffix = "Processing file..."
		sp.Start()

		for _, repo := range obj.Repositories {
			if restrictedRepos[0] != "" && !stringInSlice(repo.Name, restrictedRepos) {
				continue
			}

			if s.skipPublic && !*repo.Private {
				continue
			}

			sp.Suffix = fmt.Sprintf("Sync repository %s", repo.Name)

			logIfVerbose(fmt.Sprintf("Sync repository %s...", repo.Name))
			appendBaseToRepo(&repo, parsedFiles)
			processRepo(repo, "", s.confirmPublic)
		}

		for _, org := range obj.Organizations {
			sp.Suffix = fmt.Sprintf("Sync organization %s...", org.Name)

			logIfVerbose(fmt.Sprintf("Sync organization %s...", org.Name))
			// processOrg(org)

			for _, orgRepo := range org.Repositories {
				if restrictedRepos[0] != "" && !stringInSlice(orgRepo.Name, restrictedRepos) {
					continue
				}

				if s.skipPublic && !*orgRepo.Private {
					continue
				}

				sp.Suffix = fmt.Sprintf("Sync repository %s on organization %s...", orgRepo.Name, org.Name)

				logIfVerbose(fmt.Sprintf("Sync repo %s on organization %s...", orgRepo.Name, org.Name))
				appendBaseToRepo(&orgRepo, parsedFiles)

				for name, col := range orgRepo.Collaborators {
					fmt.Println("User:", name, ", permission: ", col.Permission)
				}
				// processRepo(orgRepo, org.Name, s.confirmPublic)
			}
		}

		sp.Stop()
	}
}

func (f *file) getFile(filePath string) *file {
	yamlFile, err := ioutil.ReadFile(filePath)
	// expand environment variables
	yamlFile = []byte(os.ExpandEnv(string(yamlFile)))
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
