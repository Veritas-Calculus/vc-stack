package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newIdentityCommand creates the identity management command.
func newIdentityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "identity",
		Aliases: []string{"iam", "auth"},
		Short:   "Manage identity and access",
		Long:    `Manage users, projects, roles, and authentication.`,
	}

	cmd.AddCommand(newIdentityUsersCommand())
	cmd.AddCommand(newIdentityProjectsCommand())
	cmd.AddCommand(newIdentityRolesCommand())

	return cmd
}

// newIdentityUsersCommand creates the users subcommand.
func newIdentityUsersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user"},
		Short:   "Manage users",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all users",
		Run:     runIdentityUsersList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		Run:   runIdentityUserCreate,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <user-id>",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		Run:   runIdentityUserDelete,
	})

	return cmd
}

// newIdentityProjectsCommand creates the projects subcommand.
func newIdentityProjectsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project", "tenant"},
		Short:   "Manage projects",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all projects",
		Run:     runIdentityProjectsList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Run:   runIdentityProjectCreate,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		Run:   runIdentityProjectDelete,
	})

	return cmd
}

// newIdentityRolesCommand creates the roles subcommand.
func newIdentityRolesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "roles",
		Aliases: []string{"role"},
		Short:   "Manage roles",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all roles",
		Run:     runIdentityRolesList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a new role",
		Run:   runIdentityRoleCreate,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <role-id>",
		Short: "Delete a role",
		Args:  cobra.ExactArgs(1),
		Run:   runIdentityRoleDelete,
	})

	return cmd
}

// --- User implementations ---

func runIdentityUsersList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/users")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	users, ok := resp["users"].([]interface{})
	if !ok || len(users) == 0 {
		fmt.Println("No users found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tUSERNAME\tEMAIL\tADMIN\tACTIVE")
	for _, item := range users {
		u, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\n",
			getString(u, "id"), getString(u, "username"),
			getString(u, "email"), u["is_admin"], u["is_active"])
	}
	_ = w.Flush()
}

func runIdentityUserCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl identity users create --username <name> --email <email> --password <pass>")
	fmt.Fprintln(os.Stderr, "Note: Use the web console or API (POST /api/v1/users) for user creation.")
}

func runIdentityUserDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/users/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("User %s deleted.\n", args[0])
}

// --- Project implementations ---

func runIdentityProjectsList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/projects")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	projects, ok := resp["projects"].([]interface{})
	if !ok || len(projects) == 0 {
		fmt.Println("No projects found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
	for _, item := range projects {
		p, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			getString(p, "id"), getString(p, "name"), getString(p, "description"))
	}
	_ = w.Flush()
}

func runIdentityProjectCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl identity projects create --name <name>")
	fmt.Fprintln(os.Stderr, "Note: Use the web console or API (POST /api/v1/projects) for project creation.")
}

func runIdentityProjectDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/projects/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Project %s deleted.\n", args[0])
}

// --- Role implementations ---

func runIdentityRolesList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/roles")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	roles, ok := resp["roles"].([]interface{})
	if !ok || len(roles) == 0 {
		fmt.Println("No roles found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
	for _, item := range roles {
		r, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			getString(r, "id"), getString(r, "name"), getString(r, "description"))
	}
	_ = w.Flush()
}

func runIdentityRoleCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl identity roles create --name <name>")
	fmt.Fprintln(os.Stderr, "Note: Use the web console or API (POST /api/v1/roles) for role creation.")
}

func runIdentityRoleDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/roles/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Role %s deleted.\n", args[0])
}
