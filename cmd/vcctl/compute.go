package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newComputeCommand creates the compute management command.
func newComputeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "compute",
		Aliases: []string{"ec2", "vm"},
		Short:   "Manage compute instances",
		Long:    `Manage virtual machine instances, flavors, and hypervisors.`,
	}

	cmd.AddCommand(newComputeInstancesCommand())
	cmd.AddCommand(newComputeFlavorsCommand())
	cmd.AddCommand(newComputeImagesCommand())
	cmd.AddCommand(newComputeSnapshotsCommand())

	return cmd
}

// newComputeInstancesCommand creates the instances subcommand.
func newComputeInstancesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "instances",
		Aliases: []string{"instance", "vms", "vm"},
		Short:   "Manage VM instances",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all instances",
		Run:     runComputeInstancesList,
	})

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new instance",
		Run:   runComputeInstanceCreate,
	}
	createCmd.Flags().String("name", "", "Instance name (required)")
	createCmd.Flags().Uint("flavor-id", 0, "Flavor ID (required)")
	createCmd.Flags().Uint("image-id", 0, "Image ID (required)")
	createCmd.Flags().StringSlice("networks", nil, "Network IDs (required)")
	createCmd.Flags().Int("root-disk-gb", 0, "Root disk size in GB")
	_ = createCmd.MarkFlagRequired("name")
	_ = createCmd.MarkFlagRequired("flavor-id")
	_ = createCmd.MarkFlagRequired("image-id")
	_ = createCmd.MarkFlagRequired("networks")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <instance-id>",
		Short: "Delete an instance",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeInstanceDelete,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "start <instance-id>",
		Short: "Start an instance",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeInstanceStart,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop <instance-id>",
		Short: "Stop an instance",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeInstanceStop,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "reboot <instance-id>",
		Short: "Reboot an instance",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeInstanceReboot,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show <instance-id>",
		Short: "Show instance details",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeInstanceShow,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "console <instance-id>",
		Short: "Get instance console URL",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeInstanceConsole,
	})

	resizeCmd := &cobra.Command{
		Use:   "resize <instance-id>",
		Short: "Resize an instance (change vCPUs/memory)",
		Long:  `Resize a running or stopped instance by specifying a new flavor or explicit vCPU/memory values.`,
		Args:  cobra.ExactArgs(1),
		Run:   runComputeInstanceResize,
	}
	resizeCmd.Flags().Uint("flavor-id", 0, "Target flavor ID")
	resizeCmd.Flags().Int("vcpus", 0, "Target vCPU count")
	resizeCmd.Flags().Int("memory-mb", 0, "Target memory in MB")
	cmd.AddCommand(resizeCmd)

	return cmd
}

// newComputeFlavorsCommand creates the flavors subcommand.
func newComputeFlavorsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "flavors",
		Aliases: []string{"flavor"},
		Short:   "Manage instance flavors",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all flavors",
		Run:     runComputeFlavorsList,
	})

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new flavor",
		Run:   runComputeFlavorCreate,
	}
	createCmd.Flags().String("name", "", "Flavor name (required)")
	createCmd.Flags().Int("vcpus", 1, "Number of vCPUs")
	createCmd.Flags().Int("memory-mb", 1024, "Memory in MB")
	createCmd.Flags().Int("disk-gb", 20, "Disk size in GB")
	_ = createCmd.MarkFlagRequired("name")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <flavor-id>",
		Short: "Delete a flavor",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeFlavorDelete,
	})

	return cmd
}

// newComputeImagesCommand creates the images subcommand.
func newComputeImagesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "images",
		Aliases: []string{"image"},
		Short:   "Manage VM images",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all images",
		Run:     runComputeImagesList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <image-id>",
		Short: "Delete an image",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeImageDelete,
	})

	return cmd
}

// newComputeSnapshotsCommand creates the snapshots subcommand.
func newComputeSnapshotsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snapshots",
		Aliases: []string{"snapshot"},
		Short:   "Manage VM snapshots",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all snapshots",
		Run:     runComputeSnapshotsList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <snapshot-id>",
		Short: "Delete a snapshot",
		Args:  cobra.ExactArgs(1),
		Run:   runComputeSnapshotDelete,
	})

	return cmd
}

// --- Instance command implementations ---

func runComputeInstancesList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/instances")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	instances, ok := resp["instances"].([]interface{})
	if !ok || len(instances) == 0 {
		fmt.Println("No instances found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tPOWER\tIP\tFLAVOR\tCREATED")
	for _, item := range instances {
		inst, _ := item.(map[string]interface{})
		flavorName := ""
		if f, ok := inst["Flavor"].(map[string]interface{}); ok {
			flavorName = getString(f, "name")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			getString(inst, "id"), getString(inst, "name"),
			getString(inst, "status"), getString(inst, "power_state"),
			getString(inst, "ip_address"), flavorName,
			getString(inst, "created_at"))
	}
	_ = w.Flush()
}

func runComputeInstanceCreate(cmd *cobra.Command, _ []string) {
	name, _ := cmd.Flags().GetString("name")
	flavorID, _ := cmd.Flags().GetUint("flavor-id")
	imageID, _ := cmd.Flags().GetUint("image-id")
	networkIDs, _ := cmd.Flags().GetStringSlice("networks")
	rootDiskGB, _ := cmd.Flags().GetInt("root-disk-gb")

	body := map[string]interface{}{
		"name":         name,
		"flavor_id":    flavorID,
		"image_id":     imageID,
		"networks":     networkIDs,
		"root_disk_gb": rootDiskGB,
	}
	c := newAPIClient()
	resp, err := c.post("/v1/instances", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Instance created successfully:")
	outputJSON(resp["instance"])
}

func runComputeInstanceDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/instances/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Instance %s deleted.\n", args[0])
}

func runComputeInstanceStart(_ *cobra.Command, args []string) {
	c := newAPIClient()
	_, err := c.post("/v1/instances/"+args[0]+"/start", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Instance %s started.\n", args[0])
}

func runComputeInstanceStop(_ *cobra.Command, args []string) {
	c := newAPIClient()
	_, err := c.post("/v1/instances/"+args[0]+"/stop", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Instance %s stopped.\n", args[0])
}

func runComputeInstanceReboot(_ *cobra.Command, args []string) {
	c := newAPIClient()
	_, err := c.post("/v1/instances/"+args[0]+"/reboot", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Instance %s rebooted.\n", args[0])
}

func runComputeInstanceShow(_ *cobra.Command, args []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/instances/" + args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputJSON(resp["instance"])
}

func runComputeInstanceConsole(_ *cobra.Command, args []string) {
	c := newAPIClient()
	resp, err := c.post("/v1/instances/"+args[0]+"/console", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputJSON(resp)
}

func runComputeInstanceResize(cmd *cobra.Command, args []string) {
	flavorID, _ := cmd.Flags().GetUint("flavor-id")
	vcpus, _ := cmd.Flags().GetInt("vcpus")
	memoryMB, _ := cmd.Flags().GetInt("memory-mb")

	if flavorID == 0 && vcpus == 0 && memoryMB == 0 {
		fmt.Fprintln(os.Stderr, "Error: specify --flavor-id or --vcpus/--memory-mb")
		os.Exit(1)
	}

	body := map[string]interface{}{}
	if flavorID > 0 {
		body["flavor_id"] = flavorID
	}
	if vcpus > 0 {
		body["vcpus"] = vcpus
	}
	if memoryMB > 0 {
		body["memory_mb"] = memoryMB
	}

	c := newAPIClient()
	_, err := c.put("/v1/instances/"+args[0], body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Instance %s resize requested.\n", args[0])
}

// --- Flavor command implementations ---

func runComputeFlavorsList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/flavors")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	flavors, ok := resp["flavors"].([]interface{})
	if !ok || len(flavors) == 0 {
		fmt.Println("No flavors found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tVCPUS\tMEMORY_MB\tDISK_GB")
	for _, item := range flavors {
		f, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%.0f\t%.0f\t%.0f\n",
			getString(f, "id"), getString(f, "name"),
			getFloat(f, "vcpus"), getFloat(f, "memory_mb"), getFloat(f, "disk_gb"))
	}
	_ = w.Flush()
}

func runComputeFlavorCreate(cmd *cobra.Command, _ []string) {
	name, _ := cmd.Flags().GetString("name")
	vcpus, _ := cmd.Flags().GetInt("vcpus")
	memoryMB, _ := cmd.Flags().GetInt("memory-mb")
	diskGB, _ := cmd.Flags().GetInt("disk-gb")

	body := map[string]interface{}{
		"name":      name,
		"vcpus":     vcpus,
		"memory_mb": memoryMB,
		"disk_gb":   diskGB,
	}
	c := newAPIClient()
	resp, err := c.post("/v1/flavors", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Flavor created:")
	outputJSON(resp["flavor"])
}

func runComputeFlavorDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/flavors/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Flavor %s deleted.\n", args[0])
}

// --- Image command implementations ---

func runComputeImagesList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/images")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	images, ok := resp["images"].([]interface{})
	if !ok || len(images) == 0 {
		fmt.Println("No images found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tFORMAT\tSIZE_GB\tSTATUS")
	for _, item := range images {
		img, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%.0f\t%s\n",
			getString(img, "id"), getString(img, "name"),
			getString(img, "format"), getFloat(img, "size_gb"),
			getString(img, "status"))
	}
	_ = w.Flush()
}

func runComputeImageDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/images/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Image %s deleted.\n", args[0])
}

// --- Snapshot command implementations ---

func runComputeSnapshotsList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/snapshots")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	snapshots, ok := resp["snapshots"].([]interface{})
	if !ok || len(snapshots) == 0 {
		fmt.Println("No snapshots found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tVOLUME\tSIZE_GB\tSTATUS\tCREATED")
	for _, item := range snapshots {
		s, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%.0f\t%s\t%s\n",
			getString(s, "id"), getString(s, "name"),
			getString(s, "volume_id"), getFloat(s, "size_gb"),
			getString(s, "status"), getString(s, "created_at"))
	}
	_ = w.Flush()
}

func runComputeSnapshotDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/snapshots/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Snapshot %s deleted.\n", args[0])
}
