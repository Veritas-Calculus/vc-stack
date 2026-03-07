package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newStorageCommand creates the storage management command.
func newStorageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "storage",
		Aliases: []string{"vol", "volume"},
		Short:   "Manage storage volumes",
		Long:    `Manage block storage volumes and volume snapshots.`,
	}

	cmd.AddCommand(newStorageVolumesCommand())
	cmd.AddCommand(newStorageSnapshotsCommand())

	return cmd
}

// newStorageVolumesCommand creates the volumes subcommand.
func newStorageVolumesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "volumes",
		Aliases: []string{"volume", "vol"},
		Short:   "Manage storage volumes",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all volumes",
		Run:     runStorageVolumesList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a new volume",
		Run:   runStorageVolumeCreate,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <volume-id>",
		Short: "Delete a volume",
		Args:  cobra.ExactArgs(1),
		Run:   runStorageVolumeDelete,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "attach <volume-id> <instance-id>",
		Short: "Attach volume to instance",
		Args:  cobra.ExactArgs(2),
		Run:   runStorageVolumeAttach,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "detach <volume-id>",
		Short: "Detach volume from instance",
		Args:  cobra.ExactArgs(1),
		Run:   runStorageVolumeDetach,
	})

	return cmd
}

// newStorageSnapshotsCommand creates the snapshots subcommand.
func newStorageSnapshotsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snapshots",
		Aliases: []string{"snapshot", "snap"},
		Short:   "Manage volume snapshots",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all snapshots",
		Run:     runStorageSnapshotsList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a volume snapshot",
		Run:   runStorageSnapshotCreate,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <snapshot-id>",
		Short: "Delete a snapshot",
		Args:  cobra.ExactArgs(1),
		Run:   runStorageSnapshotDelete,
	})

	return cmd
}

// --- Volume implementations ---

func runStorageVolumesList(_ *cobra.Command, _ []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/volumes")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	volumes, ok := resp["volumes"].([]interface{})
	if !ok || len(volumes) == 0 {
		fmt.Println("No volumes found.")
		return
	}
	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tSIZE_GB\tSTATUS\tPOOL")
	for _, item := range volumes {
		v, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%.0f\t%s\t%s\n",
			getString(v, "id"), getString(v, "name"),
			getFloat(v, "size_gb"), getString(v, "status"),
			getString(v, "rbd_pool"))
	}
	_ = w.Flush()
}

func runStorageVolumeCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl storage volumes create --name <name> --size-gb <size>")
	fmt.Fprintln(os.Stderr, "Note: Use the web console or API directly for volume creation with full options.")
}

func runStorageVolumeDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/volumes/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Volume %s deleted.\n", args[0])
}

func runStorageVolumeAttach(_ *cobra.Command, args []string) {
	c := newAPIClient()
	body := map[string]interface{}{"volume_id": args[0]}
	_, err := c.post("/v1/instances/"+args[1]+"/volumes", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Volume %s attached to instance %s.\n", args[0], args[1])
}

func runStorageVolumeDetach(_ *cobra.Command, args []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl storage volumes detach <volume-id>")
	fmt.Fprintln(os.Stderr, "Note: Detach requires the instance ID. Use the API: DELETE /api/v1/instances/<instance-id>/volumes/<volume-id>")
}

// --- Snapshot implementations ---

func runStorageSnapshotsList(_ *cobra.Command, _ []string) {
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
	fmt.Fprintln(w, "ID\tNAME\tSIZE_GB\tSTATUS\tCREATED")
	for _, item := range snapshots {
		s, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%.0f\t%s\t%s\n",
			getString(s, "id"), getString(s, "name"),
			getFloat(s, "size_gb"), getString(s, "status"),
			getString(s, "created_at"))
	}
	_ = w.Flush()
}

func runStorageSnapshotCreate(_ *cobra.Command, _ []string) {
	fmt.Fprintln(os.Stderr, "Usage: vcctl storage snapshots create --volume-id <id> --name <name>")
	fmt.Fprintln(os.Stderr, "Note: Use the web console or API directly for snapshot creation.")
}

func runStorageSnapshotDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/snapshots/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Snapshot %s deleted.\n", args[0])
}
