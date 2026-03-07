package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newNetworkCommand creates the network management command.
func newNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network",
		Aliases: []string{"net", "vpc"},
		Short:   "Manage networks and networking",
		Long:    `Manage virtual networks, subnets, routers, and security groups.`,
	}

	cmd.AddCommand(newNetworkListCommand())
	cmd.AddCommand(newNetworkSubnetsCommand())
	cmd.AddCommand(newNetworkRoutersCommand())
	cmd.AddCommand(newNetworkSecurityGroupsCommand())
	cmd.AddCommand(newNetworkFloatingIPsCommand())

	return cmd
}

func newNetworkListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all networks",
	}
	cmd.AddCommand(&cobra.Command{Use: "all", Short: "List all networks", Run: runNetworkList})
	cmd.AddCommand(&cobra.Command{Use: "create", Short: "Create a new network", Run: runNetworkCreate})
	cmd.AddCommand(&cobra.Command{Use: "delete <network-id>", Short: "Delete a network", Args: cobra.ExactArgs(1), Run: runNetworkDelete})
	return cmd
}

func newNetworkSubnetsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "subnets", Aliases: []string{"subnet"}, Short: "Manage network subnets"}
	cmd.AddCommand(&cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List all subnets", Run: runNetworkSubnetsList})
	cmd.AddCommand(&cobra.Command{Use: "create", Short: "Create a new subnet", Run: runNetworkSubnetCreate})
	cmd.AddCommand(&cobra.Command{Use: "delete <subnet-id>", Short: "Delete a subnet", Args: cobra.ExactArgs(1), Run: runNetworkSubnetDelete})
	return cmd
}

func newNetworkRoutersCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "routers", Aliases: []string{"router"}, Short: "Manage virtual routers"}
	cmd.AddCommand(&cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List all routers", Run: runNetworkRoutersList})
	cmd.AddCommand(&cobra.Command{Use: "create", Short: "Create a new router", Run: runNetworkRouterCreate})
	cmd.AddCommand(&cobra.Command{Use: "delete <router-id>", Short: "Delete a router", Args: cobra.ExactArgs(1), Run: runNetworkRouterDelete})
	return cmd
}

func newNetworkSecurityGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "security-groups", Aliases: []string{"sg", "secgroup"}, Short: "Manage security groups"}
	cmd.AddCommand(&cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List all security groups", Run: runNetworkSecurityGroupsList})
	cmd.AddCommand(&cobra.Command{Use: "create", Short: "Create a new security group", Run: runNetworkSecurityGroupCreate})
	cmd.AddCommand(&cobra.Command{Use: "delete <sg-id>", Short: "Delete a security group", Args: cobra.ExactArgs(1), Run: runNetworkSecurityGroupDelete})
	return cmd
}

func newNetworkFloatingIPsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "floating-ips", Aliases: []string{"fip", "eip"}, Short: "Manage floating IPs"}
	cmd.AddCommand(&cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List all floating IPs", Run: runNetworkFloatingIPsList})
	cmd.AddCommand(&cobra.Command{Use: "create", Short: "Allocate a new floating IP", Run: runNetworkFloatingIPCreate})
	cmd.AddCommand(&cobra.Command{Use: "delete <fip-id>", Short: "Release a floating IP", Args: cobra.ExactArgs(1), Run: runNetworkFloatingIPDelete})
	return cmd
}

// --- Network implementations ---

func runNetworkList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/networks")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	nets, ok := resp["networks"].([]interface{})
	if !ok || len(nets) == 0 {
		fmt.Println("No networks found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tCIDR\tTYPE\tEXTERNAL\tSTATUS")
	for _, item := range nets {
		n, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%s\n",
			getString(n, "id"), getString(n, "name"), getString(n, "cidr"),
			getString(n, "network_type"), n["external"], getString(n, "status"))
	}
	_ = w.Flush()
}

func runNetworkCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl network list create --name <name> --cidr <cidr>")
	fmt.Fprintln(os.Stderr, "Note: Use the web console or API directly for full network creation options.")
}

func runNetworkDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/networks/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Network %s deleted.\n", args[0])
}

// --- Subnet implementations ---

func runNetworkSubnetsList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/subnets")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	subs, ok := resp["subnets"].([]interface{})
	if !ok || len(subs) == 0 {
		fmt.Println("No subnets found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tCIDR\tGATEWAY\tNETWORK_ID")
	for _, item := range subs {
		s, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			getString(s, "id"), getString(s, "name"), getString(s, "cidr"),
			getString(s, "gateway"), getString(s, "network_id"))
	}
	_ = w.Flush()
}

func runNetworkSubnetCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl network subnets create --network-id <id> --cidr <cidr> --name <name>")
}

func runNetworkSubnetDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/subnets/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Subnet %s deleted.\n", args[0])
}

// --- Router implementations ---

func runNetworkRoutersList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/routers")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	routers, ok := resp["routers"].([]interface{})
	if !ok || len(routers) == 0 {
		fmt.Println("No routers found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tSNAT")
	for _, item := range routers {
		r, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\n",
			getString(r, "id"), getString(r, "name"),
			getString(r, "status"), r["enable_snat"])
	}
	_ = w.Flush()
}

func runNetworkRouterCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl network routers create --name <name>")
}

func runNetworkRouterDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/routers/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Router %s deleted.\n", args[0])
}

// --- Security Group implementations ---

func runNetworkSecurityGroupsList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/security-groups")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	sgs, ok := resp["security_groups"].([]interface{})
	if !ok || len(sgs) == 0 {
		fmt.Println("No security groups found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tRULES")
	for _, item := range sgs {
		sg, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%.0f\n",
			getString(sg, "id"), getString(sg, "name"),
			getString(sg, "description"), getFloat(sg, "rules_count"))
	}
	_ = w.Flush()
}

func runNetworkSecurityGroupCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl network security-groups create --name <name> --description <desc>")
}

func runNetworkSecurityGroupDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/security-groups/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Security group %s deleted.\n", args[0])
}

// --- Floating IP implementations ---

func runNetworkFloatingIPsList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/floating-ips")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fips, ok := resp["floating_ips"].([]interface{})
	if !ok || len(fips) == 0 {
		fmt.Println("No floating IPs found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tIP\tSTATUS\tINSTANCE\tNETWORK")
	for _, item := range fips {
		f, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			getString(f, "id"), getString(f, "floating_ip"),
			getString(f, "status"), getString(f, "instance_id"),
			getString(f, "network_id"))
	}
	_ = w.Flush()
}

func runNetworkFloatingIPCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl network floating-ips create --network-id <id>")
}

func runNetworkFloatingIPDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/floating-ips/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Floating IP %s released.\n", args[0])
}
