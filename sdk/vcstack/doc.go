// Package vcstack – Usage Examples
//
// This file demonstrates common usage patterns for the VC Stack Go SDK.
// It is not compiled as part of the library; it serves as documentation.
//
// # JWT Authentication (Interactive / User Login)
//
//	client := vcstack.NewClient("https://vc.example.com/api")
//	loginResp, err := client.Login(ctx, "admin", "ChangeMe123!")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Logged in, token expires in", loginResp.ExpiresIn, "seconds")
//
//	instances, err := client.Instances.List(ctx)
//
// # API Key Authentication (Service Account / Automation)
//
//	client := vcstack.NewClient("https://vc.example.com/api")
//	client.SetAPIKey("VC-AKIA-0123456789abcdef", "your-secret-key")
//	// All subsequent requests are signed with HMAC-SHA256.
//
//	instances, err := client.Instances.List(ctx)
//
// # Creating a VM
//
//	instance, err := client.Instances.Create(ctx, &vcstack.CreateInstanceRequest{
//	    Name:     "web-server-01",
//	    FlavorID: 2,
//	    ImageID:  5,
//	})
//
// # Volume Management
//
//	vol, err := client.Volumes.Create(ctx, &vcstack.CreateVolumeRequest{
//	    Name:   "data-volume",
//	    SizeGB: 100,
//	})
//	err = client.Volumes.Attach(ctx, fmt.Sprint(vol.ID), instanceID)
//
// # Network Setup
//
//	net, _ := client.Networks.Create(ctx, &vcstack.CreateNetworkRequest{
//	    Name: "app-network",
//	})
//	subnet, _ := client.Subnets.Create(ctx, &vcstack.CreateSubnetRequest{
//	    Name:      "app-subnet",
//	    CIDR:      "10.0.1.0/24",
//	    Gateway:   "10.0.1.1",
//	    NetworkID: net.ID,
//	})
//
// # Error Handling
//
//	instances, err := client.Instances.List(ctx)
//	if err != nil {
//	    if apiErr, ok := err.(*vcstack.APIError); ok {
//	        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
//	    }
//	}
package vcstack
