package main

import (
	"fmt"
	"log"

	"github.com/google/go-github/v31/github"
	"github.com/imdario/mergo"
	"github.com/jinzhu/copier"
)

type repository struct {
	Name         string `yaml:"name" json:"name,omitempty"`
	Description  string `yaml:"description" json:"description,omitempty"`
	Private      *bool  `yaml:"private" json:"private,omitempty"`
	HasIssues    *bool  `yaml:"has_issues" json:"has_issues,omitempty"`
	HasWiki      *bool  `yaml:"has_wiki" json:"has_wiki,omitempty"`
	HasPages     *bool  `yaml:"has_pages" json:"has_pages,omitempty"`
	HasProjects  *bool  `yaml:"has_projects" json:"has_projects,omitempty"`
	HasDownloads *bool  `yaml:"has_downloads" json:"has_downloads,omitempty"`

	Branches map[string]branch `yaml:"branches" json:"branches,omitempty"`

	Collaborators []collaborator `yaml:"collaborators" json:"collaborators,omitempty"`
	Hooks         []hook         `yaml:"hooks" json:"hooks,omitempty"`

	InheritFrom string `yaml:"inherit_from"`
}

type branch struct {
	Name string `yaml:"name" json:"name,omitempty"`

	Protection *protection `yaml:"protection" json:"protection,omitempty"`
}

type protection struct {
	RequiredStatusChecks       *statusCheck       `yaml:"required_status_checks" json:"required_status_checks,omitempty"`
	RequiredPullRequestReviews *requiredReviews   `yaml:"required_pull_request_reviews" json:"required_pull_request_reviews,omitempty"`
	Restrictions               *branchRestriction `yaml:"restrictions,omitempty" json:"restrictions,omitempty"`

	EnforceAdmins        *bool `yaml:"enforce_admins" json:"enforce_admins,omitempty"`
	RequireLinearHistory *bool `yaml:"required_linear_history" json:"required_linear_history,omitempty"`
	AllowForcePushes     *bool `yaml:"allow_force_pushes" json:"allow_force_pushes,omitempty"`
	AllowDeletions       *bool `yaml:"allow_deletions" json:"allow_deletions,omitempty"`
}

type statusCheck struct {
	Strict   *bool    `yaml:"strict" json:"strict,omitempty"`
	Contexts []string `yaml:"contexts" json:"contexts,omitempty"`
}

type requiredReviews struct {
	DismissStaleReviews          *bool `yaml:"dismiss_stale_reviews" json:"dismiss_stale_reviews,omitempty"`
	RequireCodeOwnerReviews      *bool `yaml:"require_code_owner_reviews" json:"require_code_owner_reviews,omitempty"`
	RequiredApprovingReviewCount *int  `yaml:"required_approving_review_count" json:"required_approving_review_count,omitempty"`
}

type branchRestriction struct {
	Apps  []string `yaml:"apps,omitempty" json:"apps,omitempty"`
	Users []string `yaml:"users,omitempty" json:"users,omitempty"`
	Teams []string `yaml:"teams,omitempty" json:"teams,omitempty"`
}

type collaborator struct {
	Name       string `yaml:"name" json:"name,omitempty"`
	Permission string `yaml:"permission" json:"permission,omitempty"`
	IsTeam     bool   `yaml:"is_team" json:"is_team,omitempty"`
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

		for branchName, branch := range repo.Branches {
			for baseBranchName, baseBranch := range d.Repository.Branches {
				if baseBranchName == branchName {
					if branch.Protection.EnforceAdmins == nil {
                        branch.Protection.EnforceAdmins = baseBranch.Protection.EnforceAdmins
                    }
                    if branch.Protection.RequireLinearHistory == nil {
                        branch.Protection.RequireLinearHistory = baseBranch.Protection.RequireLinearHistory
                    }
                    if branch.Protection.AllowForcePushes == nil {
                        branch.Protection.AllowForcePushes = baseBranch.Protection.AllowForcePushes
                    }
                    if branch.Protection.AllowDeletions == nil {
                        branch.Protection.AllowDeletions = baseBranch.Protection.AllowDeletions
                    }
				}
			}
		}

		if err := mergo.Merge(repo, d.Repository, mergo.WithAppendSlice, mergo.WithTypeCheck); err != nil {
			log.Fatalf("An error occurred: %v", err)
		}
	}
}

func processRepo(repo repository, org string, confirmPublic bool) {
	t := github.Repository{}
	copier.Copy(&t, &repo)

	// if repo is public and confirmation is enabled via `--confirm-public` show prompt to confirm
	if !*repo.Private && confirmPublic && !askForConfirmation(fmt.Sprintf("Repository %s will be set public. Are you sure? [y/n]: ", repo.Name)) {
		return
	}

	if org == "" {
		org = currentUser
	}

	_, resp, err := client.Repositories.Edit(ctx, org, *t.Name, &t)
	if err != nil && resp.StatusCode == 404 {
		if askForConfirmation(fmt.Sprintf("Oh-oh! %s does not exist on Github. Do you want to create it? [y/n]: ", repo.Name)) {
			_, _, err := client.Repositories.Create(ctx, org, &t)
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
	syncRepoHooks(repo, org)
}

func syncBranch(repo string, org string, branches map[string]branch) {
	for name, branch := range branches {
		logIfVerbose(fmt.Sprintf("Sync branch %s on repo %s\n", name, repo))

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

		_, _, err := client.Repositories.UpdateBranchProtection(ctx, org, repo, name, &t)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func syncCollaborators(repo string, org string, collaborators []collaborator) {
	logIfVerbose(fmt.Sprintf("Sync collaborators on repo %s\n", repo))
	for _, collaborator := range collaborators {
		logIfVerbose(fmt.Sprintf("Adding %s as collaborator on repo %s\n", collaborator.Name, repo))

		if collaborator.IsTeam {
			opts := github.TeamAddTeamRepoOptions{
				Permission: collaborator.Permission,
			}

			_, err := client.Teams.AddTeamRepoBySlug(ctx, org, collaborator.Name, org, repo, &opts)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			opts := github.RepositoryAddCollaboratorOptions{
				Permission: collaborator.Permission,
			}

			_, _, err := client.Repositories.AddCollaborator(ctx, org, repo, collaborator.Name, &opts)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
