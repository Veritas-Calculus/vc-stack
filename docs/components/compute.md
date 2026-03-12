# vc-compute: The Compute Node Agent

`vc-compute` is the data plane of the VC Stack. It runs on every physical host (hypervisor) and is responsible for managing the lifecycle of virtual machines (VMs) and microVMs.

## Core Responsibilities

- **VM Management**: Create, start, stop, delete, and migrate virtual machines.
- **MicroVM Support**: Dedicated support for high-performance microVMs using Firecracker.
- **Storage Operations**: Manage block storage for instances (local qcow2 or Ceph/RBD).
- **Network Agent**: Configures local OVN/OVS networking for instances.
- **Image Management**: Download, cache, and provision OS images.
- **Node Monitoring**: Report host health, CPU, memory, and disk usage to `vc-management`.

## Internal Architecture

`vc-compute` is composed of several internal services that communicate via direct function calls, ensuring high performance and low overhead.

### Key Sub-components

| Component | Responsibility | Technologies |
| :--- | :--- | :--- |
| **Orchestrator** | High-level VM lifecycle management. | Go |
| **VM Driver** | Low-level QEMU/KVM process management. | QEMU, KVM, QMP |
| **Firecracker** | MicroVM lifecycle and process isolation. | Firecracker, Jailer |
| **Network Agent** | Local bridge and port management. | OVN, OVS |
| **Storage Agent** | Local or distributed storage management. | qcow2, Ceph RBD |

## Hypervisor Support

VC Stack supports multiple hypervisor types through `vc-compute`:

### 1. KVM (QEMU)

Traditional virtualization for full-featured virtual machines.

- Full hardware virtualization.
- Support for complex device models and UEFI.
- Ideal for general-purpose workloads.

### 2. Firecracker

Minimalist microVMs designed for serverless functions and isolated containers.

- Sub-second boot times.
- Extremely low memory footprint.
- Built-in process isolation via **Jailer**.

## Storage Backends

`vc-compute` can be configured to use different storage backends for VM disks and volumes:

- **Local Storage**: Uses local `qcow2` files on the host's filesystem. Best for single-node setups or testing.
- **Ceph (RBD)**: Uses distributed Ceph block storage. Enables shared storage features like live migration and high availability.

## Networking

`vc-compute` acts as an OVN Network Agent on each host. It manages:

- **`br-int` Integration**: Connecting VM virtual interfaces (TAP devices) to the OVS integration bridge.
- **Port Binding**: Registering logical ports with the OVN controller.
- **MicroVM Networking**: Managing TAP devices and network namespaces for Firecracker.

## Heartbeat Mechanism

The compute node periodically sends a heartbeat to `vc-management`. This heartbeat includes:

- **Node Health**: Overall status (Up/Down).
- **Resource Usage**: Current CPU, RAM, and Disk consumption.
- **Instance Status**: Updated status of all VMs running on the host.

## Configuration

The compute node configuration is defined in `configs/vc-compute.yaml`. Key configuration sections include:

- `Hypervisor`: Type (kvm, firecracker), Libvirt URI.
- `Firecracker`: Paths for binary, kernel, rootfs, and jailer settings.
- `Images/Volumes`: Default backend (local/rbd) and corresponding storage paths or pool names.
- `Orchestrator`: Scheduler/Management endpoint URL.
