package main

import (
	"fmt"
	"log"

	"github.com/google/go-github/v31/github"
)

type hook struct {
	Config map[string]interface{} `yaml:"config"`
	Events []string               `yaml:"events"`
	Active bool                   `yaml:"active"`
}

func syncRepoHooks(repo repository, owner string) {
	for _, hook := range repo.Hooks {
		logIfVerbose(fmt.Sprintf("Sync hook %s into %s repository", hook.Config["url"], repo.Name))

		h := github.Hook{
			Events: hook.Events,
			Active: github.Bool(hook.Active),
			Config: hook.Config,
		}

		if hook.Config["secret"] == "" && !askForConfirmation(fmt.Sprintf("No secrets for %s are specified. Are you sure you want to continue? [y/n]: ", hook.Config["name"])) {
			continue
		}

		nh, resp, err := client.Repositories.CreateHook(ctx, owner, repo.Name, &h)
		if err != nil && resp.StatusCode == 422 && nh == nil {
			logIfVerbose(fmt.Sprintf("Hook %s already exist, updating...", hook.Config["url"]))

			existingHook := searchHook(owner, repo.Name, hook.Config)
			if existingHook != nil {
				_, _, err := client.Repositories.EditHook(ctx, owner, repo.Name, *existingHook.ID, &h)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				logIfVerbose(fmt.Sprintf("No hook with url %s found...", hook.Config["url"]))
			}
		}

	}
}

func syncOrgHooks(org organization) {
	for _, hook := range org.Hooks {
		logIfVerbose(fmt.Sprintf("Sync hook %s into %s organization", hook.Config["url"], org.Name))

		h := github.Hook{
			Events: hook.Events,
			Active: github.Bool(hook.Active),
			Config: hook.Config,
		}

		if hook.Config["secret"] == nil && !askForConfirmation(fmt.Sprintf("No secrets for %s are specified. Are you sure you want to continue? [y/n]: ", hook.Config["url"])) {
			continue
		}

		nh, resp, err := client.Organizations.CreateHook(ctx, org.Name, &h)
		if err != nil && resp.StatusCode == 422 && nh == nil {
			logIfVerbose(fmt.Sprintf("Hook %s already exist, updating...", hook.Config["url"]))

			existingHook := searchHook(org.Name, "", hook.Config)
			if existingHook != nil {
				_, _, err := client.Organizations.EditHook(ctx, org.Name, *existingHook.ID, &h)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				logIfVerbose(fmt.Sprintf("No hook with url %s found...", hook.Config["url"]))
			}
		}

	}
}

func searchHook(owner string, repo string, config map[string]interface{}) *github.Hook {
	var (
		hooks []*github.Hook
		err   error
	)
	if repo == "" {
		h, _, e := client.Organizations.ListHooks(ctx, owner, &github.ListOptions{})
		hooks = h
		err = e
	} else {
		h, _, e := client.Repositories.ListHooks(ctx, owner, repo, &github.ListOptions{})
		hooks = h
		err = e
	}

	if err != nil {
		return nil
	}
	for _, hook := range hooks {
		if hook.Config["url"] == config["url"] {
			return hook
		}
	}
	return nil
}
