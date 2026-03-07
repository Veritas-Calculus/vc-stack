import { describe, it, expect, beforeEach } from 'vitest'
import { useDataStore } from '@/lib/dataStore'

describe('dataStore', () => {
    beforeEach(() => {
        // Reset store to initial state before each test
        useDataStore.setState({
            instances: [],
            vpcs: [],
            volumes: [],
            sshKeys: [],
            flavors: [],
            snapshots: [],
            roles: [
                { id: 'r1', name: 'Administrator', roleType: 'system' },
                { id: 'r2', name: 'Viewer', roleType: 'system' }
            ],
            notices: []
        })
    })

    describe('instances', () => {
        it('adds an instance', () => {
            const { addInstance } = useDataStore.getState()
            addInstance({ projectId: 'p1', name: 'web-01', ip: '10.0.0.1', state: 'running' })
            const instances = useDataStore.getState().instances
            expect(instances).toHaveLength(1)
            expect(instances[0].name).toBe('web-01')
            expect(instances[0].state).toBe('running')
            expect(instances[0].id).toBeTruthy()
        })

        it('sets instances list', () => {
            const { setInstances } = useDataStore.getState()
            const list = [
                { id: '1', projectId: 'p1', name: 'vm-1', ip: '10.0.0.1', state: 'running' as const },
                { id: '2', projectId: 'p1', name: 'vm-2', ip: '10.0.0.2', state: 'stopped' as const }
            ]
            setInstances(list)
            expect(useDataStore.getState().instances).toHaveLength(2)
            expect(useDataStore.getState().instances[1].name).toBe('vm-2')
        })

        it('replaces existing instances on setInstances', () => {
            const { addInstance, setInstances } = useDataStore.getState()
            addInstance({ projectId: 'p1', name: 'old-vm', ip: '10.0.0.99', state: 'running' })
            expect(useDataStore.getState().instances).toHaveLength(1)

            setInstances([{ id: 'new', projectId: 'p1', name: 'new-vm', ip: '10.0.0.1', state: 'stopped' }])
            expect(useDataStore.getState().instances).toHaveLength(1)
            expect(useDataStore.getState().instances[0].name).toBe('new-vm')
        })
    })

    describe('VPCs', () => {
        it('adds a VPC with default state', () => {
            const { addVpc } = useDataStore.getState()
            addVpc({ projectId: 'p1', name: 'prod-vpc', cidr: '10.0.0.0/16' })
            const vpcs = useDataStore.getState().vpcs
            expect(vpcs).toHaveLength(1)
            expect(vpcs[0].name).toBe('prod-vpc')
            expect(vpcs[0].state).toBe('available')
        })

        it('removes a VPC', () => {
            const { addVpc } = useDataStore.getState()
            addVpc({ projectId: 'p1', name: 'to-delete', cidr: '10.1.0.0/16' })
            const id = useDataStore.getState().vpcs[0].id
            useDataStore.getState().removeVpc(id)
            expect(useDataStore.getState().vpcs).toHaveLength(0)
        })
    })

    describe('volumes', () => {
        it('adds a volume with default status', () => {
            const { addVolume } = useDataStore.getState()
            addVolume({ projectId: 'p1', name: 'data-vol', sizeGiB: 100 })
            const vols = useDataStore.getState().volumes
            expect(vols).toHaveLength(1)
            expect(vols[0].status).toBe('available')
            expect(vols[0].sizeGiB).toBe(100)
        })

        it('adds a volume with custom status', () => {
            const { addVolume } = useDataStore.getState()
            addVolume({ projectId: 'p1', name: 'attached', sizeGiB: 50, status: 'in-use' })
            expect(useDataStore.getState().volumes[0].status).toBe('in-use')
        })
    })

    describe('SSH keys', () => {
        it('adds and removes SSH keys', () => {
            const { addSSHKey, removeSSHKey } = useDataStore.getState()
            addSSHKey({ projectId: 'p1', name: 'deploy-key', publicKey: 'ssh-rsa AAAA...' })
            expect(useDataStore.getState().sshKeys).toHaveLength(1)

            const id = useDataStore.getState().sshKeys[0].id
            removeSSHKey(id)
            expect(useDataStore.getState().sshKeys).toHaveLength(0)
        })
    })

    describe('flavors', () => {
        it('sets flavors list', () => {
            const { setFlavors } = useDataStore.getState()
            setFlavors([
                { id: '1', name: 'm1.small', vcpu: 1, memoryGiB: 2 },
                { id: '2', name: 'm1.medium', vcpu: 2, memoryGiB: 4 }
            ])
            expect(useDataStore.getState().flavors).toHaveLength(2)
            expect(useDataStore.getState().flavors[0].name).toBe('m1.small')
        })

        it('adds a flavor', () => {
            const { addFlavor } = useDataStore.getState()
            addFlavor({ name: 'm1.large', vcpu: 4, memoryGiB: 8 })
            const flavors = useDataStore.getState().flavors
            expect(flavors).toHaveLength(1)
            expect(flavors[0].vcpu).toBe(4)
        })
    })

    describe('roles', () => {
        it('starts with default system roles', () => {
            expect(useDataStore.getState().roles).toHaveLength(2)
            expect(useDataStore.getState().roles[0].name).toBe('Administrator')
        })

        it('adds a custom role', () => {
            const { addRole } = useDataStore.getState()
            addRole({ name: 'Developer', roleType: 'custom' })
            expect(useDataStore.getState().roles).toHaveLength(3)
        })

        it('updates a role', () => {
            const { updateRole } = useDataStore.getState()
            updateRole('r1', { description: 'Full access' })
            expect(useDataStore.getState().roles[0].description).toBe('Full access')
        })

        it('removes a role', () => {
            const { removeRole } = useDataStore.getState()
            removeRole('r2')
            expect(useDataStore.getState().roles).toHaveLength(1)
            expect(useDataStore.getState().roles[0].name).toBe('Administrator')
        })
    })

    describe('notices', () => {
        it('marks a notice as read', () => {
            useDataStore.setState({
                notices: [
                    { id: 'n1', time: '2026-01-01', resource: 'vm-1', type: 'alert', status: 'unread' }
                ]
            })
            useDataStore.getState().markNotice('n1', 'read')
            expect(useDataStore.getState().notices[0].status).toBe('read')
        })
    })
})
