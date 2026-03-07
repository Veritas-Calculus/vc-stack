package apidocs

// openAPISpecJSON is the OpenAPI 3.0 specification in JSON format.
// This is auto-served at GET /api/v1/openapi.json.
const openAPISpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "VC Stack API",
    "description": "VC Stack Infrastructure as a Service (IaaS) API. Provides compute, network, storage, and identity management for cloud infrastructure.",
    "version": "1.0.0",
    "contact": {
      "name": "VC Stack Team"
    },
    "license": {
      "name": "Proprietary"
    }
  },
  "servers": [
    {
      "url": "/api/v1",
      "description": "Current API version"
    }
  ],
  "security": [
    {
      "BearerAuth": []
    }
  ],
  "tags": [
    {"name": "Authentication", "description": "Login, token refresh, logout"},
    {"name": "Users", "description": "User management"},
    {"name": "Roles", "description": "Role-based access control"},
    {"name": "Permissions", "description": "Permission management"},
    {"name": "Policies", "description": "IAM policy management"},
    {"name": "Projects", "description": "Multi-tenant project management"},
    {"name": "Instances", "description": "Virtual machine lifecycle"},
    {"name": "Flavors", "description": "VM instance types"},
    {"name": "Images", "description": "OS image/template management (CloudStack-style)"},
    {"name": "Volumes", "description": "Block storage volumes"},
    {"name": "Snapshots", "description": "Volume and VM snapshots"},
    {"name": "Networks", "description": "Software-defined networking (OVN)"},
    {"name": "Subnets", "description": "IP subnet management"},
    {"name": "Security Groups", "description": "Network security groups and rules"},
    {"name": "Floating IPs", "description": "Public IP address management"},
    {"name": "Routers", "description": "Virtual router management"},
    {"name": "VPCs", "description": "Virtual Private Cloud management"},
    {"name": "Ports", "description": "Network port management"},
    {"name": "Hosts", "description": "Compute node management"},
    {"name": "Zones", "description": "Availability zone management"},
    {"name": "Clusters", "description": "Cluster management"},
    {"name": "Tasks", "description": "Async task tracking"},
    {"name": "Tags", "description": "Resource tagging system"},
    {"name": "Events", "description": "Audit event logging"},
    {"name": "Quotas", "description": "Resource quota management"},
    {"name": "Notifications", "description": "Webhook/Slack notification system with retries and dead letter queue"},
    {"name": "Storage", "description": "Storage pool and volume type management"},
    {"name": "Migrations", "description": "VM live migration"},
    {"name": "SSH Keys", "description": "SSH key management"},
    {"name": "Monitoring", "description": "System metrics and health"},
    {"name": "MFA", "description": "Multi-factor authentication (TOTP)"},
    {"name": "Ports", "description": "Network port management"},
    {"name": "KMS", "description": "Key Management Service for envelope encryption"},
    {"name": "Encryption", "description": "Volume encryption and mTLS management"},
    {"name": "Compliance", "description": "Compliance audit and framework assessment"},
    {"name": "HA", "description": "High availability and fencing"},
    {"name": "DR", "description": "Disaster recovery and replication"},
    {"name": "Self-Heal", "description": "Proactive self-healing policies"},
    {"name": "CaaS", "description": "Container as a Service (Kubernetes clusters)"},
    {"name": "Bare Metal", "description": "Bare Metal as a Service (IPMI/PXE)"},
    {"name": "Object Storage", "description": "S3-compatible object storage"},
    {"name": "DNS", "description": "DNS zone and record management"},
    {"name": "Catalog", "description": "Service catalog and marketplace"},
    {"name": "Firecracker", "description": "Firecracker MicroVM management"},
    {"name": "ASNs", "description": "BGP autonomous system number management"},
    {"name": "API Discovery", "description": "API version and documentation discovery"}
  ],
  "paths": {
    "/auth/login": {
      "post": {
        "tags": ["Authentication"],
        "summary": "Login",
        "description": "Authenticate with username and password, returns JWT tokens.",
        "security": [],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["username", "password"],
                "properties": {
                  "username": {"type": "string", "example": "admin"},
                  "password": {"type": "string", "format": "password", "example": "ChangeMe123!"}
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Login successful",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "access_token": {"type": "string"},
                    "refresh_token": {"type": "string"},
                    "token_type": {"type": "string", "example": "Bearer"},
                    "expires_in": {"type": "integer"}
                  }
                }
              }
            }
          },
          "401": {"$ref": "#/components/responses/Unauthorized"}
        }
      }
    },
    "/auth/refresh": {
      "post": {
        "tags": ["Authentication"],
        "summary": "Refresh token",
        "description": "Exchange a refresh token for a new access token.",
        "security": [],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["refresh_token"],
                "properties": {
                  "refresh_token": {"type": "string"}
                }
              }
            }
          }
        },
        "responses": {
          "200": {"description": "Token refreshed"},
          "401": {"$ref": "#/components/responses/Unauthorized"}
        }
      }
    },
    "/auth/logout": {
      "post": {
        "tags": ["Authentication"],
        "summary": "Logout",
        "description": "Invalidate current session.",
        "responses": {
          "200": {"description": "Logged out"}
        }
      }
    },
    "/users": {
      "get": {
        "tags": ["Users"],
        "summary": "List users",
        "responses": {"200": {"description": "User list"}}
      },
      "post": {
        "tags": ["Users"],
        "summary": "Create user",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {"$ref": "#/components/schemas/CreateUserRequest"}
            }
          }
        },
        "responses": {
          "201": {"description": "User created"},
          "409": {"description": "User already exists"}
        }
      }
    },
    "/users/{id}": {
      "get": {
        "tags": ["Users"],
        "summary": "Get user",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {
          "200": {"description": "User details"},
          "404": {"$ref": "#/components/responses/NotFound"}
        }
      },
      "put": {
        "tags": ["Users"],
        "summary": "Update user",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "User updated"}}
      },
      "delete": {
        "tags": ["Users"],
        "summary": "Delete user",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "User deleted"}}
      }
    },
    "/instances": {
      "get": {
        "tags": ["Instances"],
        "summary": "List instances",
        "parameters": [
          {"name": "status", "in": "query", "schema": {"type": "string"}},
          {"name": "host_id", "in": "query", "schema": {"type": "string"}},
          {"name": "project_id", "in": "query", "schema": {"type": "integer"}}
        ],
        "responses": {"200": {"description": "Instance list"}}
      },
      "post": {
        "tags": ["Instances"],
        "summary": "Create instance",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {"$ref": "#/components/schemas/CreateInstanceRequest"}
            }
          }
        },
        "responses": {
          "201": {"description": "Instance created"},
          "400": {"$ref": "#/components/responses/BadRequest"},
          "409": {"description": "Quota exceeded"}
        }
      }
    },
    "/instances/{id}": {
      "get": {
        "tags": ["Instances"],
        "summary": "Get instance",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {
          "200": {"description": "Instance details"},
          "404": {"$ref": "#/components/responses/NotFound"}
        }
      },
      "delete": {
        "tags": ["Instances"],
        "summary": "Delete instance",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"202": {"description": "Deletion initiated"}}
      }
    },
    "/instances/{id}/start": {
      "post": {
        "tags": ["Instances"],
        "summary": "Start instance",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Instance started"}}
      }
    },
    "/instances/{id}/stop": {
      "post": {
        "tags": ["Instances"],
        "summary": "Stop instance",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Instance stopped"}}
      }
    },
    "/instances/{id}/reboot": {
      "post": {
        "tags": ["Instances"],
        "summary": "Reboot instance",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Instance rebooted"}}
      }
    },
    "/instances/{id}/migrate": {
      "post": {
        "tags": ["Migrations"],
        "summary": "Migrate instance",
        "description": "Initiate live or cold migration to a different host.",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "target_host": {"type": "string"},
                  "live": {"type": "boolean", "default": true}
                }
              }
            }
          }
        },
        "responses": {"202": {"description": "Migration initiated"}}
      }
    },
    "/instances/{id}/console": {
      "post": {
        "tags": ["Instances"],
        "summary": "Get VNC console URL",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Console URL"}}
      }
    },
    "/flavors": {
      "get": {
        "tags": ["Flavors"],
        "summary": "List flavors",
        "responses": {"200": {"description": "Flavor list"}}
      },
      "post": {
        "tags": ["Flavors"],
        "summary": "Create flavor",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {"$ref": "#/components/schemas/CreateFlavorRequest"}
            }
          }
        },
        "responses": {"201": {"description": "Flavor created"}}
      }
    },
    "/flavors/{id}": {
      "get": {
        "tags": ["Flavors"],
        "summary": "Get flavor",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Flavor details"}}
      },
      "delete": {
        "tags": ["Flavors"],
        "summary": "Delete flavor",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Flavor deleted"}}
      }
    },
    "/images": {
      "get": {
        "tags": ["Images"],
        "summary": "List images",
        "description": "List images with CloudStack-style filtering.",
        "parameters": [
          {"name": "visibility", "in": "query", "schema": {"type": "string", "enum": ["public", "private", "shared"]}},
          {"name": "status", "in": "query", "schema": {"type": "string"}},
          {"name": "os_type", "in": "query", "schema": {"type": "string", "enum": ["linux", "windows", "freebsd", "other"]}},
          {"name": "category", "in": "query", "schema": {"type": "string", "enum": ["user", "system", "featured", "community"]}},
          {"name": "architecture", "in": "query", "schema": {"type": "string", "enum": ["x86_64", "aarch64"]}},
          {"name": "hypervisor_type", "in": "query", "schema": {"type": "string"}},
          {"name": "disk_format", "in": "query", "schema": {"type": "string"}},
          {"name": "zone_id", "in": "query", "schema": {"type": "string"}},
          {"name": "search", "in": "query", "schema": {"type": "string"}},
          {"name": "bootable", "in": "query", "schema": {"type": "string", "enum": ["true"]}}
        ],
        "responses": {"200": {"description": "Image list with total count"}}
      },
      "post": {
        "tags": ["Images"],
        "summary": "Create image",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {"$ref": "#/components/schemas/CreateImageRequest"}
            }
          }
        },
        "responses": {"201": {"description": "Image created"}}
      }
    },
    "/images/{id}": {
      "get": {
        "tags": ["Images"],
        "summary": "Get image",
        "description": "Returns image details with instance usage count.",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Image details with instance_count"}}
      },
      "put": {
        "tags": ["Images"],
        "summary": "Update image metadata",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Image updated"}}
      },
      "delete": {
        "tags": ["Images"],
        "summary": "Delete image",
        "description": "Fails if image is protected or in use by active instances.",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {
          "200": {"description": "Image deleted"},
          "403": {"description": "Image is protected"},
          "409": {"description": "Image in use by active instances"}
        }
      }
    },
    "/images/upload": {
      "post": {
        "tags": ["Images"],
        "summary": "Upload image file",
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "required": ["file"],
                "properties": {
                  "file": {"type": "string", "format": "binary"},
                  "name": {"type": "string"},
                  "disk_format": {"type": "string"}
                }
              }
            }
          }
        },
        "responses": {"202": {"description": "Upload accepted, processing async"}}
      }
    },
    "/images/{id}/import": {
      "post": {
        "tags": ["Images"],
        "summary": "Import image from URL or RBD",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "source_url": {"type": "string", "format": "uri"},
                  "file_path": {"type": "string"},
                  "rbd_pool": {"type": "string"},
                  "rbd_image": {"type": "string"},
                  "rbd_snap": {"type": "string"}
                }
              }
            }
          }
        },
        "responses": {"202": {"description": "Import initiated"}}
      }
    },
    "/volumes": {
      "get": {
        "tags": ["Volumes"],
        "summary": "List volumes",
        "responses": {"200": {"description": "Volume list"}}
      },
      "post": {
        "tags": ["Volumes"],
        "summary": "Create volume",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {"$ref": "#/components/schemas/CreateVolumeRequest"}
            }
          }
        },
        "responses": {"201": {"description": "Volume created"}}
      }
    },
    "/volumes/{id}": {
      "get": {
        "tags": ["Volumes"],
        "summary": "Get volume",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Volume details"}}
      },
      "delete": {
        "tags": ["Volumes"],
        "summary": "Delete volume",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Volume deleted"}}
      }
    },
    "/networks": {
      "get": {
        "tags": ["Networks"],
        "summary": "List networks",
        "responses": {"200": {"description": "Network list"}}
      },
      "post": {
        "tags": ["Networks"],
        "summary": "Create network",
        "responses": {"201": {"description": "Network created"}}
      }
    },
    "/networks/{id}": {
      "get": {
        "tags": ["Networks"],
        "summary": "Get network",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Network details"}}
      },
      "put": {
        "tags": ["Networks"],
        "summary": "Update network",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Network updated"}}
      },
      "delete": {
        "tags": ["Networks"],
        "summary": "Delete network",
        "parameters": [{"$ref": "#/components/parameters/IdParam"}],
        "responses": {"200": {"description": "Network deleted"}}
      }
    },
    "/subnets": {
      "get": {"tags": ["Subnets"], "summary": "List subnets", "responses": {"200": {"description": "Subnet list"}}},
      "post": {"tags": ["Subnets"], "summary": "Create subnet", "responses": {"201": {"description": "Subnet created"}}}
    },
    "/subnets/{id}": {
      "get": {"tags": ["Subnets"], "summary": "Get subnet", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Subnet details"}}},
      "put": {"tags": ["Subnets"], "summary": "Update subnet", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Subnet updated"}}},
      "delete": {"tags": ["Subnets"], "summary": "Delete subnet", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Subnet deleted"}}}
    },
    "/security-groups": {
      "get": {"tags": ["Security Groups"], "summary": "List security groups", "responses": {"200": {"description": "Security group list"}}},
      "post": {"tags": ["Security Groups"], "summary": "Create security group", "responses": {"201": {"description": "Security group created"}}}
    },
    "/security-groups/{id}": {
      "get": {"tags": ["Security Groups"], "summary": "Get security group", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Security group details"}}},
      "put": {"tags": ["Security Groups"], "summary": "Update security group", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Security group updated"}}},
      "delete": {"tags": ["Security Groups"], "summary": "Delete security group", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Security group deleted"}}}
    },
    "/security-group-rules": {
      "get": {"tags": ["Security Groups"], "summary": "List security group rules", "responses": {"200": {"description": "Rule list"}}},
      "post": {"tags": ["Security Groups"], "summary": "Create security group rule", "responses": {"201": {"description": "Rule created"}}}
    },
    "/floating-ips": {
      "get": {"tags": ["Floating IPs"], "summary": "List floating IPs", "responses": {"200": {"description": "Floating IP list"}}},
      "post": {"tags": ["Floating IPs"], "summary": "Allocate floating IP", "responses": {"201": {"description": "Floating IP allocated"}}}
    },
    "/floating-ips/{id}": {
      "get": {"tags": ["Floating IPs"], "summary": "Get floating IP", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Floating IP details"}}},
      "put": {"tags": ["Floating IPs"], "summary": "Associate/dissociate floating IP", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Floating IP updated"}}},
      "delete": {"tags": ["Floating IPs"], "summary": "Release floating IP", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Floating IP released"}}}
    },
    "/routers": {
      "get": {"tags": ["Routers"], "summary": "List routers", "responses": {"200": {"description": "Router list"}}},
      "post": {"tags": ["Routers"], "summary": "Create router", "responses": {"201": {"description": "Router created"}}}
    },
    "/routers/{id}": {
      "get": {"tags": ["Routers"], "summary": "Get router", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Router details"}}},
      "put": {"tags": ["Routers"], "summary": "Update router", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Router updated"}}},
      "delete": {"tags": ["Routers"], "summary": "Delete router", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Router deleted"}}}
    },
    "/routers/{id}/add-interface": {
      "post": {"tags": ["Routers"], "summary": "Add router interface", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Interface added"}}}
    },
    "/routers/{id}/remove-interface": {
      "post": {"tags": ["Routers"], "summary": "Remove router interface", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Interface removed"}}}
    },
    "/vpcs": {
      "get": {"tags": ["VPCs"], "summary": "List VPCs", "responses": {"200": {"description": "VPC list"}}},
      "post": {"tags": ["VPCs"], "summary": "Create VPC", "responses": {"201": {"description": "VPC created"}}}
    },
    "/vpcs/{id}": {
      "get": {"tags": ["VPCs"], "summary": "Get VPC", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "VPC details"}}},
      "delete": {"tags": ["VPCs"], "summary": "Delete VPC", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "VPC deleted"}}}
    },
    "/hosts": {
      "get": {"tags": ["Hosts"], "summary": "List hosts", "responses": {"200": {"description": "Host list"}}},
      "post": {"tags": ["Hosts"], "summary": "Register host", "responses": {"201": {"description": "Host registered"}}}
    },
    "/hosts/{id}": {
      "get": {"tags": ["Hosts"], "summary": "Get host", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Host details"}}},
      "delete": {"tags": ["Hosts"], "summary": "Remove host", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Host removed"}}}
    },
    "/zones": {
      "get": {"tags": ["Zones"], "summary": "List zones", "responses": {"200": {"description": "Zone list"}}},
      "post": {"tags": ["Zones"], "summary": "Create zone", "responses": {"201": {"description": "Zone created"}}}
    },
    "/zones/{id}": {
      "get": {"tags": ["Zones"], "summary": "Get zone", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Zone details"}}},
      "put": {"tags": ["Zones"], "summary": "Update zone", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Zone updated"}}},
      "delete": {"tags": ["Zones"], "summary": "Delete zone", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Zone deleted"}}}
    },
    "/clusters": {
      "get": {"tags": ["Clusters"], "summary": "List clusters", "responses": {"200": {"description": "Cluster list"}}},
      "post": {"tags": ["Clusters"], "summary": "Create cluster", "responses": {"201": {"description": "Cluster created"}}}
    },
    "/clusters/{id}": {
      "get": {"tags": ["Clusters"], "summary": "Get cluster", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Cluster details"}}},
      "put": {"tags": ["Clusters"], "summary": "Update cluster", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Cluster updated"}}},
      "delete": {"tags": ["Clusters"], "summary": "Delete cluster", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Cluster deleted"}}}
    },
    "/tasks": {
      "get": {
        "tags": ["Tasks"],
        "summary": "List tasks",
        "parameters": [
          {"name": "status", "in": "query", "schema": {"type": "string", "enum": ["pending", "running", "completed", "failed", "cancelled"]}},
          {"name": "type", "in": "query", "schema": {"type": "string"}},
          {"name": "resource_type", "in": "query", "schema": {"type": "string"}},
          {"name": "resource_id", "in": "query", "schema": {"type": "string"}}
        ],
        "responses": {"200": {"description": "Task list"}}
      }
    },
    "/tasks/{id}": {
      "get": {"tags": ["Tasks"], "summary": "Get task", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Task details"}}},
      "delete": {"tags": ["Tasks"], "summary": "Delete task", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Task deleted"}}}
    },
    "/tasks/{id}/cancel": {
      "post": {"tags": ["Tasks"], "summary": "Cancel task", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Task cancelled"}}}
    },
    "/tags": {
      "get": {
        "tags": ["Tags"],
        "summary": "List tags",
        "parameters": [
          {"name": "resource_type", "in": "query", "schema": {"type": "string"}},
          {"name": "key", "in": "query", "schema": {"type": "string"}}
        ],
        "responses": {"200": {"description": "Tag list"}}
      }
    },
    "/tags/{resourceType}/{resourceId}": {
      "get": {"tags": ["Tags"], "summary": "Get resource tags", "responses": {"200": {"description": "Tags for resource"}}},
      "post": {
        "tags": ["Tags"],
        "summary": "Set resource tags",
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "tags": {"type": "object", "additionalProperties": {"type": "string"}}
                }
              }
            }
          }
        },
        "responses": {"200": {"description": "Tags set"}}
      },
      "delete": {"tags": ["Tags"], "summary": "Delete all resource tags", "responses": {"200": {"description": "Tags deleted"}}}
    },
    "/events": {
      "get": {
        "tags": ["Events"],
        "summary": "List audit events",
        "parameters": [
          {"name": "resource_type", "in": "query", "schema": {"type": "string"}},
          {"name": "action", "in": "query", "schema": {"type": "string"}},
          {"name": "user_id", "in": "query", "schema": {"type": "integer"}}
        ],
        "responses": {"200": {"description": "Event list"}}
      }
    },
    "/quotas/tenants/{tenant_id}": {
      "get": {"tags": ["Quotas"], "summary": "Get tenant quota", "responses": {"200": {"description": "Quota details"}}},
      "put": {"tags": ["Quotas"], "summary": "Update tenant quota", "responses": {"200": {"description": "Quota updated"}}},
      "delete": {"tags": ["Quotas"], "summary": "Reset tenant quota to defaults", "responses": {"200": {"description": "Quota reset"}}}
    },
    "/quotas/tenants/{tenant_id}/usage": {
      "get": {"tags": ["Quotas"], "summary": "Get tenant resource usage", "responses": {"200": {"description": "Usage details"}}}
    },
    "/notifications/channels": {
      "get": {"tags": ["Notifications"], "summary": "List notification channels", "responses": {"200": {"description": "Channel list"}}},
      "post": {"tags": ["Notifications"], "summary": "Create notification channel", "responses": {"201": {"description": "Channel created"}}}
    },
    "/notifications/channels/{id}": {
      "get": {"tags": ["Notifications"], "summary": "Get channel", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Channel details with subscriptions"}}},
      "put": {"tags": ["Notifications"], "summary": "Update channel", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Channel updated"}}},
      "delete": {"tags": ["Notifications"], "summary": "Delete channel", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Channel deleted"}}}
    },
    "/notifications/channels/{id}/test": {
      "post": {"tags": ["Notifications"], "summary": "Send test notification", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Test notification sent"}}}
    },
    "/notifications/subscriptions": {
      "get": {"tags": ["Notifications"], "summary": "List subscriptions", "responses": {"200": {"description": "Subscription list"}}},
      "post": {"tags": ["Notifications"], "summary": "Create subscription", "responses": {"201": {"description": "Subscription created"}}}
    },
    "/notifications/logs": {
      "get": {"tags": ["Notifications"], "summary": "List notification logs", "description": "Query delivery logs with status filter (sent, failed, dead_letter).", "parameters": [{"name": "status", "in": "query", "schema": {"type": "string", "enum": ["sent", "failed", "dead_letter"]}}, {"name": "channel_id", "in": "query", "schema": {"type": "integer"}}], "responses": {"200": {"description": "Notification log list"}}}
    },
    "/notifications/dead-letters": {
      "get": {"tags": ["Notifications"], "summary": "List dead letter entries", "description": "Notifications that failed after all retry attempts.", "responses": {"200": {"description": "Dead letter list"}}},
      "delete": {"tags": ["Notifications"], "summary": "Purge all dead letters", "responses": {"200": {"description": "All entries purged"}}}
    },
    "/notifications/dead-letters/{id}/retry": {
      "post": {"tags": ["Notifications"], "summary": "Retry dead letter", "description": "Re-queue a failed notification for delivery.", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Re-queued for delivery"}}}
    },
    "/migrations": {
      "get": {"tags": ["Migrations"], "summary": "List migrations", "responses": {"200": {"description": "Migration list"}}}
    },
    "/migrations/{id}": {
      "get": {"tags": ["Migrations"], "summary": "Get migration", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Migration details"}}}
    },
    "/migrations/{id}/cancel": {
      "post": {"tags": ["Migrations"], "summary": "Cancel migration", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Migration cancelled"}}}
    },
    "/ssh-keys": {
      "get": {"tags": ["SSH Keys"], "summary": "List SSH keys", "responses": {"200": {"description": "SSH key list"}}},
      "post": {"tags": ["SSH Keys"], "summary": "Create SSH key", "responses": {"201": {"description": "SSH key created"}}}
    },
    "/ssh-keys/{id}": {
      "delete": {"tags": ["SSH Keys"], "summary": "Delete SSH key", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "SSH key deleted"}}}
    },
    "/snapshots": {
      "get": {"tags": ["Snapshots"], "summary": "List snapshots", "responses": {"200": {"description": "Snapshot list"}}},
      "post": {"tags": ["Snapshots"], "summary": "Create snapshot", "responses": {"201": {"description": "Snapshot created"}}}
    },
    "/snapshots/{id}": {
      "get": {"tags": ["Snapshots"], "summary": "Get snapshot", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Snapshot details"}}},
      "delete": {"tags": ["Snapshots"], "summary": "Delete snapshot", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Snapshot deleted"}}}
    },
    "/monitoring/status": {
      "get": {
        "tags": ["Monitoring"],
        "summary": "Component status",
        "responses": {"200": {"description": "Component status overview"}}
      }
    },
    "/auth/mfa/setup": {
      "post": {"tags": ["MFA"], "summary": "Begin MFA setup", "description": "Generate TOTP secret and QR code for authenticator app.", "responses": {"200": {"description": "TOTP secret, QR URI, and recovery codes"}}}
    },
    "/auth/mfa/verify": {
      "post": {"tags": ["MFA"], "summary": "Verify MFA code", "description": "Verify TOTP code to complete MFA setup or login challenge.", "responses": {"200": {"description": "MFA verified"}, "401": {"description": "Invalid code"}}}
    },
    "/auth/mfa/disable": {
      "post": {"tags": ["MFA"], "summary": "Disable MFA", "responses": {"200": {"description": "MFA disabled"}}}
    },
    "/auth/mfa/status": {
      "get": {"tags": ["MFA"], "summary": "Get MFA status", "responses": {"200": {"description": "MFA enabled/disabled status"}}}
    },
    "/auth/mfa/recovery-codes": {
      "post": {"tags": ["MFA"], "summary": "Regenerate recovery codes", "responses": {"200": {"description": "New recovery codes"}}}
    },
    "/ports": {
      "get": {"tags": ["Ports"], "summary": "List network ports", "responses": {"200": {"description": "Port list"}}},
      "post": {"tags": ["Ports"], "summary": "Create port", "responses": {"201": {"description": "Port created"}}}
    },
    "/ports/{id}": {
      "get": {"tags": ["Ports"], "summary": "Get port", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Port details"}}},
      "delete": {"tags": ["Ports"], "summary": "Delete port", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Port deleted"}}}
    },
    "/kms/keys": {
      "get": {"tags": ["KMS"], "summary": "List encryption keys", "responses": {"200": {"description": "Key list"}}},
      "post": {"tags": ["KMS"], "summary": "Create encryption key", "responses": {"201": {"description": "Key created"}}}
    },
    "/kms/keys/{id}": {
      "get": {"tags": ["KMS"], "summary": "Get key", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Key details"}}},
      "delete": {"tags": ["KMS"], "summary": "Delete key", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Key deleted"}}}
    },
    "/kms/keys/{id}/rotate": {
      "post": {"tags": ["KMS"], "summary": "Rotate key", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Key rotated"}}}
    },
    "/encryption/volumes/{id}/encrypt": {
      "post": {"tags": ["Encryption"], "summary": "Encrypt volume", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"202": {"description": "Encryption initiated"}}}
    },
    "/encryption/mtls/certificates": {
      "get": {"tags": ["Encryption"], "summary": "List mTLS certificates", "responses": {"200": {"description": "Certificate list"}}},
      "post": {"tags": ["Encryption"], "summary": "Issue mTLS certificate", "responses": {"201": {"description": "Certificate issued"}}}
    },
    "/compliance/assessments": {
      "get": {"tags": ["Compliance"], "summary": "List compliance assessments", "responses": {"200": {"description": "Assessment list"}}},
      "post": {"tags": ["Compliance"], "summary": "Run compliance assessment", "description": "Assess against SOC 2, ISO 27001, PCI DSS, GDPR, or HIPAA.", "responses": {"201": {"description": "Assessment started"}}}
    },
    "/compliance/reports": {
      "get": {"tags": ["Compliance"], "summary": "List compliance reports", "responses": {"200": {"description": "Report list"}}}
    },
    "/ha/groups": {
      "get": {"tags": ["HA"], "summary": "List HA groups", "responses": {"200": {"description": "HA group list"}}},
      "post": {"tags": ["HA"], "summary": "Create HA group", "responses": {"201": {"description": "HA group created"}}}
    },
    "/ha/groups/{id}/failover": {
      "post": {"tags": ["HA"], "summary": "Trigger failover", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"202": {"description": "Failover initiated"}}}
    },
    "/dr/plans": {
      "get": {"tags": ["DR"], "summary": "List DR plans", "responses": {"200": {"description": "DR plan list"}}},
      "post": {"tags": ["DR"], "summary": "Create DR plan", "responses": {"201": {"description": "DR plan created"}}}
    },
    "/dr/plans/{id}/drill": {
      "post": {"tags": ["DR"], "summary": "Execute DR drill", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"202": {"description": "DR drill initiated"}}}
    },
    "/self-heal/policies": {
      "get": {"tags": ["Self-Heal"], "summary": "List self-heal policies", "responses": {"200": {"description": "Policy list"}}},
      "post": {"tags": ["Self-Heal"], "summary": "Create self-heal policy", "responses": {"201": {"description": "Policy created"}}}
    },
    "/self-heal/incidents": {
      "get": {"tags": ["Self-Heal"], "summary": "List self-heal incidents", "responses": {"200": {"description": "Incident list"}}}
    },
    "/caas/clusters": {
      "get": {"tags": ["CaaS"], "summary": "List Kubernetes clusters", "responses": {"200": {"description": "Cluster list"}}},
      "post": {"tags": ["CaaS"], "summary": "Create Kubernetes cluster", "responses": {"201": {"description": "Cluster creation initiated"}}}
    },
    "/caas/clusters/{id}": {
      "get": {"tags": ["CaaS"], "summary": "Get K8s cluster", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Cluster details with node pools"}}},
      "delete": {"tags": ["CaaS"], "summary": "Delete K8s cluster", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"202": {"description": "Deletion initiated"}}}
    },
    "/caas/clusters/{id}/kubeconfig": {
      "get": {"tags": ["CaaS"], "summary": "Download kubeconfig", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Kubeconfig YAML"}}}
    },
    "/baremetal/nodes": {
      "get": {"tags": ["Bare Metal"], "summary": "List bare metal nodes", "responses": {"200": {"description": "Node list"}}},
      "post": {"tags": ["Bare Metal"], "summary": "Register bare metal node", "responses": {"201": {"description": "Node registered"}}}
    },
    "/baremetal/nodes/{id}": {
      "get": {"tags": ["Bare Metal"], "summary": "Get bare metal node", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Node details with hardware inventory"}}},
      "delete": {"tags": ["Bare Metal"], "summary": "Decommission node", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Node decommissioned"}}}
    },
    "/baremetal/nodes/{id}/power": {
      "post": {"tags": ["Bare Metal"], "summary": "IPMI power action", "description": "Power on/off/reset via IPMI.", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Power action executed"}}}
    },
    "/baremetal/nodes/{id}/provision": {
      "post": {"tags": ["Bare Metal"], "summary": "Provision OS via PXE", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"202": {"description": "Provisioning initiated"}}}
    },
    "/object-storage/buckets": {
      "get": {"tags": ["Object Storage"], "summary": "List S3 buckets", "responses": {"200": {"description": "Bucket list"}}},
      "post": {"tags": ["Object Storage"], "summary": "Create bucket", "responses": {"201": {"description": "Bucket created"}}}
    },
    "/object-storage/buckets/{name}": {
      "get": {"tags": ["Object Storage"], "summary": "Get bucket details", "responses": {"200": {"description": "Bucket details"}}},
      "delete": {"tags": ["Object Storage"], "summary": "Delete bucket", "responses": {"200": {"description": "Bucket deleted"}}}
    },
    "/dns/zones": {
      "get": {"tags": ["DNS"], "summary": "List DNS zones", "responses": {"200": {"description": "Zone list"}}},
      "post": {"tags": ["DNS"], "summary": "Create DNS zone", "responses": {"201": {"description": "Zone created"}}}
    },
    "/dns/zones/{id}/records": {
      "get": {"tags": ["DNS"], "summary": "List DNS records", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "Record list"}}},
      "post": {"tags": ["DNS"], "summary": "Create DNS record", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"201": {"description": "Record created"}}}
    },
    "/catalog/services": {
      "get": {"tags": ["Catalog"], "summary": "List catalog services", "responses": {"200": {"description": "Service list"}}},
      "post": {"tags": ["Catalog"], "summary": "Create catalog service", "responses": {"201": {"description": "Service created"}}}
    },
    "/catalog/requests": {
      "get": {"tags": ["Catalog"], "summary": "List provisioning requests", "responses": {"200": {"description": "Request list"}}},
      "post": {"tags": ["Catalog"], "summary": "Submit provisioning request", "responses": {"201": {"description": "Request submitted"}}}
    },
    "/firecracker/vms": {
      "get": {"tags": ["Firecracker"], "summary": "List Firecracker MicroVMs", "responses": {"200": {"description": "MicroVM list"}}},
      "post": {"tags": ["Firecracker"], "summary": "Create MicroVM", "description": "Launch a Firecracker microVM with kernel and rootfs.", "responses": {"201": {"description": "MicroVM created"}}}
    },
    "/firecracker/vms/{id}": {
      "get": {"tags": ["Firecracker"], "summary": "Get MicroVM", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "MicroVM details"}}},
      "delete": {"tags": ["Firecracker"], "summary": "Delete MicroVM", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "MicroVM deleted"}}}
    },
    "/firecracker/vms/{id}/snapshot": {
      "post": {"tags": ["Firecracker"], "summary": "Create snapshot", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"201": {"description": "Snapshot created"}}}
    },
    "/firecracker/pool/stats": {
      "get": {"tags": ["Firecracker"], "summary": "Get pool stats", "description": "Pre-warmed MicroVM pool statistics.", "responses": {"200": {"description": "Pool stats"}}}
    },
    "/firecracker/functions": {
      "post": {"tags": ["Firecracker"], "summary": "Invoke function", "description": "FaaS: invoke a function using a pre-warmed MicroVM from the pool.", "responses": {"200": {"description": "Function result"}}}
    },
    "/asns": {
      "get": {"tags": ["ASNs"], "summary": "List ASNs", "responses": {"200": {"description": "ASN list"}}},
      "post": {"tags": ["ASNs"], "summary": "Create ASN", "responses": {"201": {"description": "ASN created"}}}
    },
    "/asns/{id}": {
      "get": {"tags": ["ASNs"], "summary": "Get ASN", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "ASN details"}}},
      "put": {"tags": ["ASNs"], "summary": "Update ASN", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "ASN updated"}}},
      "delete": {"tags": ["ASNs"], "summary": "Delete ASN", "parameters": [{"$ref": "#/components/parameters/IdParam"}], "responses": {"200": {"description": "ASN deleted"}}}
    }
  },
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      }
    },
    "parameters": {
      "IdParam": {
        "name": "id",
        "in": "path",
        "required": true,
        "schema": {"type": "string"},
        "description": "Resource ID (numeric or UUID)"
      }
    },
    "schemas": {
      "CreateUserRequest": {
        "type": "object",
        "required": ["username", "email", "password"],
        "properties": {
          "username": {"type": "string", "minLength": 3},
          "email": {"type": "string", "format": "email"},
          "password": {"type": "string", "format": "password", "minLength": 8},
          "role": {"type": "string", "enum": ["admin", "member", "viewer"]}
        }
      },
      "CreateInstanceRequest": {
        "type": "object",
        "required": ["name", "flavor_id", "image_id"],
        "properties": {
          "name": {"type": "string"},
          "flavor_id": {"type": "integer"},
          "image_id": {"type": "integer"},
          "network_id": {"type": "integer"},
          "ssh_key": {"type": "string"},
          "user_data": {"type": "string"},
          "enable_tpm": {"type": "boolean"}
        }
      },
      "CreateFlavorRequest": {
        "type": "object",
        "required": ["name", "vcpus", "ram_mb", "disk_gb"],
        "properties": {
          "name": {"type": "string"},
          "vcpus": {"type": "integer", "minimum": 1},
          "ram_mb": {"type": "integer", "minimum": 128},
          "disk_gb": {"type": "integer", "minimum": 1},
          "is_public": {"type": "boolean", "default": true}
        }
      },
      "CreateImageRequest": {
        "type": "object",
        "required": ["name"],
        "properties": {
          "name": {"type": "string"},
          "description": {"type": "string"},
          "disk_format": {"type": "string", "enum": ["qcow2", "raw", "vmdk", "iso"]},
          "os_type": {"type": "string", "enum": ["linux", "windows", "freebsd", "other"]},
          "os_version": {"type": "string", "example": "ubuntu-22.04"},
          "architecture": {"type": "string", "enum": ["x86_64", "aarch64"], "default": "x86_64"},
          "hypervisor_type": {"type": "string", "default": "kvm"},
          "category": {"type": "string", "enum": ["user", "system", "featured", "community"], "default": "user"},
          "visibility": {"type": "string", "enum": ["public", "private", "shared"], "default": "private"},
          "min_disk": {"type": "integer"},
          "min_ram": {"type": "integer"},
          "protected": {"type": "boolean"},
          "bootable": {"type": "boolean", "default": true}
        }
      },
      "CreateVolumeRequest": {
        "type": "object",
        "required": ["name", "size_gb"],
        "properties": {
          "name": {"type": "string"},
          "size_gb": {"type": "integer", "minimum": 1},
          "rbd_pool": {"type": "string"}
        }
      },
      "APIError": {
        "type": "object",
        "description": "Standardized error response. All errors include a machine-readable code, human-readable message, and backward-compatible error field.",
        "required": ["code", "message", "error"],
        "properties": {
          "code": {"type": "string", "description": "Machine-readable error code (e.g., ResourceNotFound, QuotaExceeded)", "example": "ResourceNotFound"},
          "message": {"type": "string", "description": "Human-readable error message", "example": "instance not found"},
          "error": {"type": "string", "description": "Same as message (backward compatibility)", "example": "instance not found"},
          "detail": {"type": "string", "description": "Additional context about the error", "example": "id: 42"},
          "field": {"type": "string", "description": "Field that caused the error (validation errors)"},
          "request_id": {"type": "string", "description": "Request ID for tracing", "example": "20260305-abc123"}
        }
      },
      "ErrorCodes": {
        "type": "string",
        "description": "All available error codes",
        "enum": [
          "AuthenticationRequired", "InvalidCredentials", "TokenExpired", "TokenInvalid",
          "AccessDenied", "RateLimitExceeded",
          "ValidationFailed", "InvalidParameter", "MissingRequired", "InvalidFormat",
          "ResourceNotFound", "ResourceAlreadyExists", "ResourceInUse", "ResourceProtected", "StateConflict",
          "QuotaExceeded", "LimitExceeded",
          "InternalError", "ServiceUnavailable", "OperationFailed", "DatabaseError", "StorageError",
          "InstanceNotFound", "InvalidInstanceState", "FlavorNotFound", "ImageNotFound", "ImageProtected", "ImageInUse",
          "NetworkNotFound", "SubnetNotFound", "CIDRConflict", "IPAddressExhausted",
          "VolumeNotFound", "VolumeInUse", "SnapshotNotFound",
          "HostNotFound", "NoHostAvailable", "MigrationFailed"
        ]
      }
    },
    "responses": {
      "BadRequest": {
        "description": "Invalid request (ValidationFailed, InvalidParameter, MissingRequired)",
        "content": {
          "application/json": {
            "schema": {"$ref": "#/components/schemas/APIError"}
          }
        }
      },
      "Unauthorized": {
        "description": "Authentication required (AuthenticationRequired, TokenInvalid, TokenExpired)",
        "content": {
          "application/json": {
            "schema": {"$ref": "#/components/schemas/APIError"}
          }
        }
      },
      "NotFound": {
        "description": "Resource not found (ResourceNotFound, InstanceNotFound, ImageNotFound, etc.)",
        "content": {
          "application/json": {
            "schema": {"$ref": "#/components/schemas/APIError"}
          }
        }
      }
    }
  }
}`

// openAPISpecYAML placeholder - redirect to JSON.
const openAPISpecYAML = `# VC Stack OpenAPI Specification
# For the full spec, use the JSON endpoint: /api/v1/openapi.json
# This YAML version is a simplified reference.
openapi: "3.0.3"
info:
  title: VC Stack API
  version: "1.0.0"
  description: >
    VC Stack IaaS API. See /api/v1/openapi.json for the complete specification.
servers:
  - url: /api/v1
    description: Current API version
`
