package network

import (
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"gorm.io/gorm"
)

// IPAllocation tracks allocated IPs per subnet.
type IPAllocation struct {
	ID        uint   `gorm:"primaryKey"`
	SubnetID  string `gorm:"not null;uniqueIndex:subnet_ip"`
	IP        string `gorm:"not null;uniqueIndex:subnet_ip"`
	PortID    string `gorm:"index"`
	CreatedAt time.Time
}

func (IPAllocation) TableName() string { return "net_ip_allocations" }

// IPAM provides simple IP allocation from subnet CIDR ranges.
type IPAM struct {
	db  *gorm.DB
	opt IPAMOptions
}

func NewIPAM(db *gorm.DB, opt IPAMOptions) *IPAM { return &IPAM{db: db, opt: opt} }

// Allocate returns the next free IP in the subnet (skipping network/broadcast).
func (i *IPAM) Allocate(subnet *Subnet, portID string) (string, error) {
	_, ipnet, err := net.ParseCIDR(subnet.CIDR)
	if err != nil {
		return "", err
	}
	start := subnet.AllocationStart
	end := subnet.AllocationEnd
	// Iterate usable range.
	begin := firstUsableIP(ipnet, start)
	// apply reserved-first offset.
	for k := 0; k < i.opt.ReservedFirst; k++ {
		begin = nextIP(begin)
	}
	// compute last usable cutoff considering reserved_last.
	cutoff := lastUsableIP(ipnet, end)
	for k := 0; k < i.opt.ReservedLast; k++ {
		cutoff = prevIP(cutoff)
	}
	for ip := begin; ipnet.Contains(ip) && compareIP(ip, cutoff) <= 0; ip = nextIP(ip) {
		if end != "" && compareIP(ip, net.ParseIP(end)) > 0 {
			break
		}
		if isNetworkOrBroadcast(ip, ipnet) {
			continue
		}
		if i.opt.ReserveGateway && subnet.Gateway != "" && ip.Equal(net.ParseIP(subnet.Gateway)) {
			continue
		}
		var count int64
		i.db.Model(&IPAllocation{}).Where("subnet_id = ? AND ip = ?", subnet.ID, ip.String()).Count(&count)
		if count == 0 {
			rec := IPAllocation{SubnetID: subnet.ID, IP: ip.String(), PortID: portID}
			if err := i.db.Create(&rec).Error; err != nil {
				return "", err
			}
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no free IPs in subnet %s", subnet.ID)
}

// Release frees an IP if allocated.
func (i *IPAM) Release(subnetID, ip, portID string) error {
	q := i.db.Where("subnet_id = ? AND ip = ?", subnetID, ip)
	if portID != "" {
		q = q.Where("port_id = ?", portID)
	}
	return q.Delete(&IPAllocation{}).Error
}

// Helpers.
func firstUsableIP(n *net.IPNet, start string) net.IP {
	if start != "" {
		if ip := net.ParseIP(start); ip != nil {
			return ip
		}
	}
	ip := n.IP.Mask(n.Mask)
	ip = nextIP(ip) // skip network address
	return ip
}

func isNetworkOrBroadcast(ip net.IP, n *net.IPNet) bool {
	if !n.Contains(ip) {
		return false
	}
	// network.
	if ip.Equal(n.IP.Mask(n.Mask)) {
		return true
	}
	// broadcast for IPv4.
	if v4 := ip.To4(); v4 != nil {
		bcast := make(net.IP, len(n.IP.To4()))
		for i := 0; i < 4; i++ {
			bcast[i] = n.IP.To4()[i] | ^n.Mask[i]
		}
		return v4.Equal(bcast)
	}
	return false
}

func nextIP(ip net.IP) net.IP {
	x := big.NewInt(0).SetBytes(ip.To16())
	x.Add(x, big.NewInt(1))
	b := x.Bytes()
	if len(b) < net.IPv6len {
		pad := make([]byte, net.IPv6len-len(b))
		b = append(pad, b...)
	}
	return net.IP(b)
}

func prevIP(ip net.IP) net.IP {
	x := big.NewInt(0).SetBytes(ip.To16())
	x.Sub(x, big.NewInt(1))
	b := x.Bytes()
	if len(b) < net.IPv6len {
		pad := make([]byte, net.IPv6len-len(b))
		b = append(pad, b...)
	}
	return net.IP(b)
}

func lastUsableIP(n *net.IPNet, end string) net.IP {
	if end != "" {
		if ip := net.ParseIP(end); ip != nil {
			return ip
		}
	}
	// broadcast - 1 for IPv4; for IPv6, just use last address in prefix minus 1.
	if v4 := n.IP.To4(); v4 != nil {
		bcast := make(net.IP, len(v4))
		for i := 0; i < 4; i++ {
			bcast[i] = n.IP.To4()[i] | ^n.Mask[i]
		}
		return prevIP(bcast)
	}
	// IPv6 last address in subnet minus 1.
	// Build max address by OR with inverted mask.
	base := n.IP.To16()
	inv := make([]byte, net.IPv6len)
	for i := 0; i < net.IPv6len; i++ {
		inv[i] = ^n.Mask[i]
	}
	last := make([]byte, net.IPv6len)
	for i := 0; i < net.IPv6len; i++ {
		last[i] = base[i] | inv[i]
	}
	return prevIP(net.IP(last))
}

// compareIP compares ip a and b; returns -1,0,1.
func compareIP(a, b net.IP) int {
	if b == nil {
		return -1
	}
	aa := a.To16()
	bb := b.To16()
	return strings.Compare(string(aa), string(bb))
}
