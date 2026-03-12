# VC Stack Component Documentation

This directory contains detailed documentation for the core components of VC Stack.

## Components

| Component | Description | Documentation |
| :--- | :--- | :--- |
| **vc-management** | The centralized management plane (control plane). | [Management Documentation](management.md) |
| **vc-compute** | The compute node agent (data plane). | [Compute Documentation](compute.md) |
| **vcctl** | The unified command-line interface. | [CLI Documentation](vcctl.md) |

## Component Relationships

```mermaid
graph TD
    User([User]) -->|CLI/Web| MGMT[vc-management]
    MGMT -->|HTTP/gRPC| COMP1[vc-compute 1]
    MGMT -->|HTTP/gRPC| COMPN[vc-compute N]
    MGMT <--> DB[(PostgreSQL)]
    COMP1 <--> Storage[(Ceph/RBD)]
    COMPN <--> Storage
```

For a high-level overview of the entire system, see the [Architecture Guide](../ARCHITECTURE.md).
