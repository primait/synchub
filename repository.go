package main

import (
	"fmt"
	"log"

	"github.com/google/go-github/v31/github"
	"github.com/imdario/mergo"
	"github.com/jinzhu/copier"
)

type repository struct {
	Name         string `yaml:"name"`
	Description  string `yaml:"description"`
	Private      bool   `yaml:"private"`
	HasIssues    bool   `yaml:"has_issues"`
	HasWiki      bool   `yaml:"has_wiki"`
	HasPages     bool   `yaml:"has_pages"`
	HasProjects  bool   `yaml:"has_projects"`
	HasDownloads bool   `yaml:"has_downloads"`

	Branches []branch `yaml:"branches"`

	Collaborators []collaborator `yaml:"collaborators"`

	InheritFrom string `yaml:"inherit_from"`
}

type branch struct {
	Name string `yaml:"name"`

	Protection protection `yaml:"protection"`
}

type protection struct {
	RequiredStatusChecks       *statusCheck      `yaml:"required_status_checks"`
	RequiredPullRequestReviews *requiredReviews  `yaml:"required_pull_request_reviews"`
	Restrictions               branchRestriction `yaml:"restrictions,omitempty"`

	EnforceAdmins        *bool `yaml:"enforce_admins"`
	RequireLinearHistory *bool `yaml:"required_linear_history"`
	AllowForcePushes     *bool `yaml:"allow_force_pushes"`
	AllowDeletions       *bool `yaml:"allow_deletions"`
}

type statusCheck struct {
	Strict   *bool     `yaml:"strict"`
	Contexts *[]string `yaml:"contexts"`
}

type requiredReviews struct {
	DismissStaleReviews          *bool `yaml:"dismiss_stale_reviews"`
	RequireCodeOwnerReviews      *bool `yaml:"require_code_owner_reviews"`
	RequiredApprovingReviewCount *int  `yaml:"required_approving_review_count"`
}

type branchRestriction struct {
	Apps  *[]string `yaml:"apps,omitempty"`
	Users *[]string `yaml:"users,omitempty"`
	Teams *[]string `yaml:"teams,omitempty"`
}

type collaborator struct {
	Name       string `yaml:"name"`
	Permission string `yaml:"permission"`
}

func appendBaseToRepo(repo *repository, parsedFiles []*file) {
	if repo.InheritFrom != "" {
		var d *base
	I:
		for _, obj := range parsedFiles {
			for _, base := range obj.Bases {
				if repo.InheritFrom == base.Name {
					d = &base
					break I
				}
			}
		}
		if d == nil {
			log.Fatalf("Error searching \"%s\" base defined in %s repo", repo.InheritFrom, repo.Name)
		}
		if err := mergo.Merge(repo, d.Repository, mergo.WithOverride); err != nil {
			log.Fatalf("An error occurred: %v", err)
		}
	}
}

func processRepo(repo repository, org string, confirmPublic bool) {
	t := github.Repository{}
	copier.Copy(&t, &repo)

	// if repo is public e confirmation is enabled via `--confirm-public` show prompt to confirm
	if !repo.Private && confirmPublic && !askForConfirmation(fmt.Sprintf("Repository %s will be set public. Are you sure? [y/n]: ", repo.Name)) {
		return
	}

	_, resp, err := editRepo(org, &t)
	if err != nil && resp.StatusCode == 404 {
		if askForConfirmation(fmt.Sprintf("Oh-oh! %s does not exist on Github. Do you want to create it? [y/n]: ", repo.Name)) {
			_, _, err := createRepo(org, &t)
			if err != nil {
				log.Fatal(err)
			}
			logIfVerbose(fmt.Sprintf("Successfully updated repo: %v\n", repo.Name))
		}
	} else if err != nil {
		log.Fatal(err)
	} else {
		logIfVerbose(fmt.Sprintf("Successfully updated repo: %v\n", repo.Name))
	}

	syncBranch(repo.Name, org, repo.Branches)
	syncCollaborators(repo.Name, org, repo.Collaborators)
}

func syncBranch(repo string, org string, branches []branch) {
	for _, branch := range branches {
		logIfVerbose(fmt.Sprintf("Sync branch %s on repo %s\n", branch.Name, repo))

		// RequiredStatusChecks
		a := github.RequiredStatusChecks{}
		copier.Copy(&a, &branch.Protection.RequiredStatusChecks)

		// RequiredPullRequestReviews
		b := github.PullRequestReviewsEnforcementRequest{}
		copier.Copy(&b, &branch.Protection.RequiredPullRequestReviews)

		// Restrictions
		c := github.BranchRestrictionsRequest{}
		copier.Copy(&c, &branch.Protection.Restrictions)

		t := github.ProtectionRequest{RequiredStatusChecks: &a, RequiredPullRequestReviews: &b, Restrictions: &c}
		copier.Copy(&t, &branch.Protection)

		_, _, err := editRepoBranches(org, repo, branch.Name, &t)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func syncCollaborators(repo string, org string, collaborators []collaborator) {
	logIfVerbose(fmt.Sprintf("Sync collaborators on repo %s\n", repo))
	for _, collaborator := range collaborators {
		logIfVerbose(fmt.Sprintf("%s added as collaborator on repo %s\n", collaborator.Name, repo))

		if org == "" {
			org = currentUser
		}

		opts := github.RepositoryAddCollaboratorOptions{
			Permission: collaborator.Permission,
		}

		_, _, err := client.Repositories.AddCollaborator(ctx, org, repo, collaborator.Name, &opts)
		if err != nil {
			log.Fatal(err)
		}
	}
}
