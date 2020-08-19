package main

import (
	"fmt"
	"log"

	"github.com/google/go-github/v31/github"
	"github.com/jinzhu/copier"
)

type organization struct {
	Name    string   `yaml:"name"`
	Profile *profile `yaml:"profile,omitempty"`

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
	Id          string
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Maintainers []string `yaml:"maintainers,omitempty"`
	RepoNames   []string `yaml:"repo_names,omitempty"`
	ParentTeam  string   `yaml:"parent_team,omitempty"`
	Privacy     string   `yaml:"privacy,omitempty"`

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
	currentTeams, _, _ := client.Teams.ListTeams(ctx, org.Name, &github.ListOptions{})

	deletedTeams := deletedTeams(org.Teams, currentTeams)
	for _, d := range deletedTeams {
		logIfVerbose(fmt.Sprintf("Delete team %s from %s", *d.Name, org.Name))
		_, err := client.Teams.DeleteTeamBySlug(ctx, org.Name, *d.Slug)
		if err != nil {
			fmt.Println("An error occurred:", err)
		}
	}

	for _, team := range org.Teams {
		logIfVerbose(fmt.Sprintf("Sync team %s", team.Name))

		t := github.NewTeam{}
		copier.Copy(&t, &team)

		teamId, teamErr := getParentTeamId(org, slug(team.ParentTeam))
		if teamErr == nil {
			t.ParentTeamID = &teamId
		}

		_, _, err := client.Teams.CreateTeam(ctx, org.Name, t)
		if err != nil {
			logIfVerbose(fmt.Sprintf("Update existing team %s", team.Name))
			_, _, err := client.Teams.EditTeamBySlug(ctx, org.Name, slug(team.Name), t, false)
			if err != nil {
				fmt.Println("An error occurred:", err)
			}

			currentMembers, _, err := client.Teams.ListTeamMembersBySlug(ctx, org.Name, slug(team.Name), &github.TeamListTeamMembersOptions{})
			if err != nil {
				fmt.Println("An error occurred:", err)
				continue
			}

			deletedMembers := deletedMembers(team.Members, currentMembers)
			for _, m := range deletedMembers {
				logIfVerbose(fmt.Sprintf("Delete member %s from team %s", *m.Login, team.Name))
				_, err := client.Teams.RemoveTeamMembershipBySlug(ctx, org.Name, slug(team.Name), *m.Login)
				if err != nil {
					fmt.Println("An error occurred:", err)
				}
			}
		}

		for _, member := range team.Members {
			logIfVerbose(fmt.Sprintf("Grant permission %s to user %s on team %s", member.Role, member.Name, team.Name))

			var role = github.TeamAddTeamMembershipOptions{
				Role: member.Role,
			}
			_, _, err := client.Teams.AddTeamMembershipBySlug(ctx, org.Name, slug(team.Name), member.Name, &role)
			if err != nil {
				fmt.Println("An error occurred:", err)
			}
		}
	}
}

func deletedTeams(current []team, teams []*github.Team) []github.Team {
	var diff []github.Team
	for _, t := range teams {
		var f = false
	J:
		for _, c := range current {
			if c.Name == *t.Name {
				f = true
				break J
			}
		}
		if !f {
			diff = append(diff, *t)
		}
	}
	return diff
}

func deletedMembers(current []teamMember, users []*github.User) []github.User {
	var diff []github.User
	for _, t := range users {
		var f = false
	J:
		for _, c := range current {
			if c.Name == *t.Login {
				f = true
				break J
			}
		}
		if !f {
			diff = append(diff, *t)
		}
	}
	return diff
}

func getParentTeamId(org organization, slug string) (int64, error) {
	team, _, err := client.Teams.GetTeamBySlug(ctx, org.Name, slug)
	if err != nil {
		return -1, err
	}
	return *team.ID, nil
}
