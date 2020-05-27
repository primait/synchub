package main

import (
	"fmt"
	"log"

	"github.com/google/go-github/v31/github"
	"github.com/jinzhu/copier"
)

type organization struct {
	Name     string  `yaml:"name"`
	Profile profile `yaml:"profile"`

	Repositories []repository `yaml:"repositories"`
	Teams        []team       `yaml:"teams"`
	Hooks        []hook       `yaml:"hooks"`
}

type profile struct {
	BillingEmail string `yaml:"billing_email"`
	Company      string `yaml:"company"`
	Email        string `yaml:"email"`
	Location     string `yaml:"location"`
	Description  string `yaml:"description"`

	MembersCanCreateRepos        bool   `yaml:"members_can_create_repositories"`
	MembersCanCreatePublicRepos  bool   `yaml:"members_can_create_public_repositories"`
	MembersCanCreatePrivateRepos bool   `yaml:"members_can_create_private_repositories"`
	DefaultRepoPermission        string `yaml:"default_repository_permission"`
}

type team struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description,omitempty"`
	Maintainers  []string `yaml:"maintainers,omitempty"`
	RepoNames    []string `yaml:"repo_names,omitempty"`
	ParentTeamID *int64   `yaml:"parent_team_id,omitempty"`
	Privacy      string   `yaml:"privacy,omitempty"`

	Members []teamMember `yaml:"members,omitempty"`
}

type teamMember struct {
	Name string `yaml:"name,omitempty"`
	Role string `yaml:"role,omitempty"`
}

func processOrg(org organization) {
	t := github.Organization{}
	copier.Copy(&t, &org.Profile)

	_, resp, err := client.Organizations.Edit(ctx, org.Name, &t)
	if err != nil {
		fmt.Printf("%v", resp.Response.Body)
		log.Fatal(err)
	}

	syncOrgTeams(org)
	syncOrgHooks(org)
}

func syncOrgTeams(org organization) {
	for _, team := range org.Teams {
		logIfVerbose(fmt.Sprintf("Sync team %s", team.Name))
		t := github.NewTeam{}
		copier.Copy(&t, &team)

		_, _, err := client.Teams.CreateTeam(ctx, org.Name, t)
		if err != nil {
			logIfVerbose(fmt.Sprintf("Update existing team %s", team.Name))
			_, _, editErr := client.Teams.EditTeamBySlug(ctx, org.Name, slug(team.Name), t, false)
			if editErr != nil {
				log.Fatal(editErr)
			}
		}

		for _, member := range team.Members {
			logIfVerbose(fmt.Sprintf("Grant permission %s to user %s on team %s", member.Role, member.Name, team.Name))

			var role = github.TeamAddTeamMembershipOptions{
				Role: member.Role,
			}
			_, _, err := client.Teams.AddTeamMembershipBySlug(ctx, org.Name, slug(team.Name), member.Name, &role)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
