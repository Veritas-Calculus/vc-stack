package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newClusterCommand creates the cluster management command.
func newClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster",
		Aliases: []string{"node", "host"},
		Short:   "Manage cluster nodes and resources",
		Long:    `Manage compute nodes, hosts, zones, clusters, and infrastructure resources.`,
	}

	cmd.AddCommand(newClusterNodesCommand())
	cmd.AddCommand(newClusterZonesCommand())
	cmd.AddCommand(newClusterClustersCommand())

	return cmd
}

// ── Nodes ──

func newClusterNodesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "nodes",
		Aliases: []string{"node", "hosts", "host"},
		Short:   "Manage cluster nodes",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all cluster nodes",
		RunE:    runClusterNodesList,
	})

	showCmd := &cobra.Command{
		Use:   "show <node-id>",
		Short: "Show node details",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterNodeShow,
	}
	cmd.AddCommand(showCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable <node-id>",
		Short: "Enable a node for scheduling",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterNodeEnable,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "disable <node-id>",
		Short: "Disable a node from scheduling",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterNodeDisable,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "maintenance <node-id>",
		Short: "Put node in maintenance mode",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterNodeMaintenance,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <node-id>",
		Short: "Remove a node from cluster",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterNodeDelete,
	})

	return cmd
}

func runClusterNodesList(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	data, err := c.get("/v1/hosts")
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	if output == "json" {
		outputJSON(data)
		return nil
	}

	hosts, ok := data["hosts"].([]interface{})
	if !ok {
		fmt.Println("No nodes found.")
		return nil
	}

	w := newTabWriter()
	fmt.Fprintln(w, "UUID\tNAME\tSTATUS\tSTATE\tIP\tCPU\tRAM (MB)\tDISK (GB)\tZONE\tCLUSTER")
	for _, h := range hosts {
		m, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%.0f\t%.0f\t%.0f\t%s\t%s\n",
			getString(m, "uuid"),
			getString(m, "name"),
			getString(m, "status"),
			getString(m, "resource_state"),
			getString(m, "ip_address"),
			getFloat(m, "cpu_cores"),
			getFloat(m, "ram_mb"),
			getFloat(m, "disk_gb"),
			getString(m, "zone_id"),
			getString(m, "cluster_id"),
		)
	}
	w.Flush()
	return nil
}

func runClusterNodeShow(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	data, err := c.get("/v1/hosts/" + args[0])
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}
	outputJSON(data)
	return nil
}

func runClusterNodeEnable(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	_, err := c.post("/v1/hosts/"+args[0]+"/enable", nil)
	if err != nil {
		return fmt.Errorf("failed to enable node: %w", err)
	}
	fmt.Printf("Node %s enabled.\n", args[0])
	return nil
}

func runClusterNodeDisable(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	_, err := c.post("/v1/hosts/"+args[0]+"/disable", nil)
	if err != nil {
		return fmt.Errorf("failed to disable node: %w", err)
	}
	fmt.Printf("Node %s disabled.\n", args[0])
	return nil
}

func runClusterNodeMaintenance(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	_, err := c.post("/v1/hosts/"+args[0]+"/maintenance", nil)
	if err != nil {
		return fmt.Errorf("failed to set maintenance: %w", err)
	}
	fmt.Printf("Node %s set to maintenance mode.\n", args[0])
	return nil
}

func runClusterNodeDelete(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	if err := c.delete("/v1/hosts/" + args[0]); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}
	fmt.Printf("Node %s deleted.\n", args[0])
	return nil
}

// ── Zones ──

func newClusterZonesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "zones",
		Aliases: []string{"zone", "az"},
		Short:   "Manage availability zones",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all zones",
		RunE:    runClusterZonesList,
	})

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new zone",
		RunE:  runClusterZoneCreate,
	}
	createCmd.Flags().String("name", "", "Zone name (required)")
	createCmd.Flags().String("type", "core", "Zone type: core or edge")
	createCmd.Flags().String("network-type", "Advanced", "Network type: Basic or Advanced")
	_ = createCmd.MarkFlagRequired("name")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "show <zone-id>",
		Short: "Show zone details",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterZoneShow,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <zone-id>",
		Short: "Delete a zone",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterZoneDelete,
	})

	return cmd
}

func runClusterZonesList(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	data, err := c.get("/v1/zones")
	if err != nil {
		return fmt.Errorf("failed to list zones: %w", err)
	}

	if output == "json" {
		outputJSON(data)
		return nil
	}

	zones, ok := data["zones"].([]interface{})
	if !ok {
		fmt.Println("No zones found.")
		return nil
	}

	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tNETWORK\tALLOCATION")
	for _, z := range zones {
		m, ok := z.(map[string]interface{})
		if !ok {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			getString(m, "id"),
			getString(m, "name"),
			getString(m, "type"),
			getString(m, "network_type"),
			getString(m, "allocation"),
		)
	}
	w.Flush()
	return nil
}

func runClusterZoneCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	zoneType, _ := cmd.Flags().GetString("type")
	networkType, _ := cmd.Flags().GetString("network-type")

	c := newAPIClient()
	data, err := c.post("/v1/zones", map[string]string{
		"name":         name,
		"type":         zoneType,
		"network_type": networkType,
	})
	if err != nil {
		return fmt.Errorf("failed to create zone: %w", err)
	}
	fmt.Printf("Zone created.\n")
	outputJSON(data)
	return nil
}

func runClusterZoneShow(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	data, err := c.get("/v1/zones/" + args[0])
	if err != nil {
		return fmt.Errorf("failed to get zone: %w", err)
	}
	outputJSON(data)
	return nil
}

func runClusterZoneDelete(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	if err := c.delete("/v1/zones/" + args[0]); err != nil {
		return fmt.Errorf("failed to delete zone: %w", err)
	}
	fmt.Printf("Zone %s deleted.\n", args[0])
	return nil
}

// ── Clusters ──

func newClusterClustersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clusters",
		Aliases: []string{"cl"},
		Short:   "Manage compute clusters",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all clusters",
		RunE:    runClusterClustersList,
	})

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new cluster",
		RunE:  runClusterClusterCreate,
	}
	createCmd.Flags().String("name", "", "Cluster name (required)")
	createCmd.Flags().String("zone-id", "", "Zone ID to assign")
	createCmd.Flags().String("hypervisor", "kvm", "Hypervisor type")
	createCmd.Flags().String("description", "", "Description")
	_ = createCmd.MarkFlagRequired("name")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "show <cluster-id>",
		Short: "Show cluster details",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterClusterShow,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <cluster-id>",
		Short: "Delete a cluster",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterClusterDelete,
	})

	return cmd
}

func runClusterClustersList(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	data, err := c.get("/v1/clusters")
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	if output == "json" {
		outputJSON(data)
		return nil
	}

	clusters, ok := data["clusters"].([]interface{})
	if !ok {
		fmt.Println("No clusters found.")
		return nil
	}

	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tZONE\tHYPERVISOR\tALLOCATION\tDESCRIPTION")
	for _, cl := range clusters {
		m, ok := cl.(map[string]interface{})
		if !ok {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			getString(m, "id"),
			getString(m, "name"),
			getString(m, "zone_id"),
			getString(m, "hypervisor_type"),
			getString(m, "allocation"),
			getString(m, "description"),
		)
	}
	w.Flush()
	return nil
}

func runClusterClusterCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	hypervisor, _ := cmd.Flags().GetString("hypervisor")
	desc, _ := cmd.Flags().GetString("description")

	body := map[string]interface{}{
		"name":            name,
		"hypervisor_type": hypervisor,
		"description":     desc,
	}
	if zoneID != "" {
		body["zone_id"] = zoneID
	}

	c := newAPIClient()
	data, err := c.post("/v1/clusters", body)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	if output == "json" {
		outputJSON(data)
	} else {
		fmt.Println("Cluster created.")
		outputJSON(data)
	}
	return nil
}

func runClusterClusterShow(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	data, err := c.get("/v1/clusters/" + args[0])
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	outputJSON(data)
	return nil
}

func runClusterClusterDelete(cmd *cobra.Command, args []string) error {
	c := newAPIClient()
	if err := c.delete("/v1/clusters/" + args[0]); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}
	fmt.Printf("Cluster %s deleted.\n", args[0])
	return nil
}

// Silence unused variable warning for os package.
var _ = os.Stdout
