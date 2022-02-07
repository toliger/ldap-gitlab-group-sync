package main

import (
	"log"
	"github.com/xanzy/go-gitlab"
  "github.com/go-ldap/ldap"
  "sync"
  "encoding/json"
  "fmt"
  "os"
)


type Group_Sync struct {
  LDAPGroupName string
  AccessLevel int
}

func Connect() (*ldap.Conn) {
    l, err := ldap.DialURL(fmt.Sprintf("ldap://%s:389", os.Getenv("GITLAB_LDAP_GROUP_MAPPER_LDAP_FQDN")))
    if err != nil {
        log.Fatal(err)
    }
    l.Bind(os.Getenv("GITLAB_LDAP_GROUP_MAPPER_LDAP_BINDUSERNAME"), os.Getenv("GITLAB_LDAP_GROUP_MAPPER_LDAP_BINDPASSWORD"))

    return l
}

func GitlabConnect() (*gitlab.Client) {
	git, err := gitlab.NewClient(os.Getenv("GITLAB_LDAP_GROUP_MAPPER_GITLAB_TOKEN"), gitlab.WithBaseURL(os.Getenv("GITLAB_LDAP_GROUP_MAPPER_GITLAB_DOMAIN")))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
  return git
}

func main() {
	git :=  GitlabConnect() // Gitlab context
  l := Connect() // LDAP context

  group_options := &gitlab.ListGroupsOptions{
    AllAvailable:         gitlab.Bool(true),
	}

	groups, _, err := git.Groups.ListGroups(group_options)
	
  if err != nil {
		log.Fatal(err)
	}

  // List Top level Gitlab groups
  for _, e := range groups {
	  sggroups, _, err := git.Groups.ListDescendantGroups(e.ID, nil)

	  if err != nil {
		  log.Fatal(err)
	  }

    var wg sync.WaitGroup

    wg.Add(len(sggroups))

    // List all Gitlabsub groups
    for _, se := range sggroups {
      go func(se *gitlab.Group) {
        defer wg.Done()

    	  settings, _, err := git.GroupVariables.GetVariable(se.ID, "LDAP_GITLAB_SYNC_SETTINGS")


        // Check if the group should be synchronized
    	  if err == nil {
          var groups_syncs []Group_Sync
          json.Unmarshal([]byte(settings.Value), &groups_syncs)
    
          // Get users from groups
          for _, ldap_group := range groups_syncs {
            log.Print(ldap_group.LDAPGroupName, ldap_group.AccessLevel)
            searchRequest := ldap.NewSearchRequest(
              os.Getenv("GITLAB_LDAP_GROUP_MAPPER_LDAP_BASEDN"),
              ldap.ScopeWholeSubtree,
              ldap.NeverDerefAliases,
              0,
              0,
              false,
              fmt.Sprintf(os.Getenv("GITLAB_LDAP_GROUP_MAPPER_LDAP_FILTER"), ldap_group.LDAPGroupName),
              []string{},
              nil,
            )
            sr, err := l.Search(searchRequest)
            if err != nil {
              log.Fatal(err)
            }

            // Add Users to the group members
            for _, u := range sr.Entries {
              user := u.GetAttributeValue("sAMAccountName")
              log.Print(user)
    
              
    	        options := gitlab.ListUsersOptions{
    		        Username: &user,
    	        }
    
    	        usrs, _, err := git.Users.ListUsers(&options)
              if err != nil {
                panic(err)
              }
    
              if len(usrs) != 0 {
    
                al := gitlab.AccessLevelValue(ldap_group.AccessLevel)
                memberoption := &gitlab.AddGroupMemberOptions{
                  UserID:         gitlab.Int(usrs[0].ID),
                  AccessLevel:    &al,
                }
                git.GroupMembers.AddGroupMember(se.ID, memberoption)
              }
            }
          }
        }
      }(se)
	  }
    wg.Wait()
  }
}
