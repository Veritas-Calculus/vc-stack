package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newImageCommand creates the top-level image management command.
// This provides a convenient shortcut: `vcctl image list` instead of `vcctl compute images list`.
func newImageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "image",
		Aliases: []string{"images"},
		Short:   "Manage OS images",
		Long:    `Manage operating system images used for provisioning virtual machines.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all images",
		Run:     runImageList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show <image-id>",
		Short: "Show image details",
		Args:  cobra.ExactArgs(1),
		Run:   runImageShow,
	})

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Register a new image",
		Run:   runImageCreate,
	}
	createCmd.Flags().String("name", "", "Image name (required)")
	createCmd.Flags().String("format", "qcow2", "Disk format (qcow2, raw, iso)")
	createCmd.Flags().String("url", "", "Image source URL for download")
	createCmd.Flags().String("visibility", "private", "Visibility: public, private")
	createCmd.Flags().Int("min-disk", 0, "Minimum required disk (GB)")
	createCmd.Flags().Int("min-ram", 0, "Minimum required RAM (MB)")
	_ = createCmd.MarkFlagRequired("name")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <image-id>",
		Short: "Delete an image",
		Args:  cobra.ExactArgs(1),
		Run:   runImageDelete,
	})

	return cmd
}

// --- Image command implementations ---

func runImageList(_ *cobra.Command, _ []string) {
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
	fmt.Fprintln(w, "ID\tNAME\tFORMAT\tSIZE_GB\tVISIBILITY\tSTATUS")
	for _, item := range images {
		img, _ := item.(map[string]interface{})
		fmt.Fprintf(w, "%s\t%s\t%s\t%.0f\t%s\t%s\n",
			getString(img, "id"), getString(img, "name"),
			getString(img, "format"), getFloat(img, "size_gb"),
			getString(img, "visibility"), getString(img, "status"))
	}
	_ = w.Flush()
}

func runImageShow(_ *cobra.Command, args []string) {
	c := newAPIClient()
	resp, err := c.get("/v1/images/" + args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputJSON(resp["image"])
}

func runImageCreate(cmd *cobra.Command, _ []string) {
	name, _ := cmd.Flags().GetString("name")
	format, _ := cmd.Flags().GetString("format")
	url, _ := cmd.Flags().GetString("url")
	visibility, _ := cmd.Flags().GetString("visibility")
	minDisk, _ := cmd.Flags().GetInt("min-disk")
	minRAM, _ := cmd.Flags().GetInt("min-ram")

	body := map[string]interface{}{
		"name":        name,
		"disk_format": format,
		"visibility":  visibility,
	}
	if url != "" {
		body["url"] = url
	}
	if minDisk > 0 {
		body["min_disk"] = minDisk
	}
	if minRAM > 0 {
		body["min_ram"] = minRAM
	}

	c := newAPIClient()
	resp, err := c.post("/v1/images", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Image registered:")
	outputJSON(resp["image"])
}

func runImageDelete(_ *cobra.Command, args []string) {
	c := newAPIClient()
	if err := c.delete("/v1/images/" + args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Image %s deleted.\n", args[0])
}
