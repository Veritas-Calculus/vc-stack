package network

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// IPAM provides robust IP allocation from subnet CIDR ranges.
type IPAM struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewIPAM(db *gorm.DB, logger *zap.Logger) *IPAM {
	return &IPAM{db: db, logger: logger}
}

// Allocate returns the next free IP in the subnet using a pessimistic row-level lock.
func (i *IPAM) Allocate(ctx context.Context, subnet *Subnet, portID string) (string, error) {
	var allocatedIP string

	// Use a transaction to ensure atomicity.
	err := i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Pessimistic Lock: Lock the Subnet row.
		// Use dialect-aware locking (Postgres/MySQL use FOR UPDATE, SQLite ignores it).
		var s Subnet
		query := tx.Where("id = ?", subnet.ID)

		// Detection: SQLite doesn't support FOR UPDATE
		dialect := tx.Dialector.Name()
		if dialect != "sqlite" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}

		if err := query.First(&s).Error; err != nil {
			return fmt.Errorf("failed to acquire lock on subnet: %w", err)
		}

		_, ipnet, err := net.ParseCIDR(s.CIDR)
		if err != nil {
			return fmt.Errorf("invalid CIDR %s: %w", s.CIDR, err)
		}

		begin := firstUsableIP(ipnet, s.AllocationStart)
		cutoff := lastUsableIP(ipnet, s.AllocationEnd)

		// 2. Count usable range size for random offset
		totalUsable := i.countRange(begin, cutoff, ipnet, s.Gateway)
		if totalUsable <= 0 {
			return errors.New("no usable IPs in range")
		}

		// Pick a random starting point to distribute allocations
		var offset int
		if totalUsable > 0 {
			b := make([]byte, 8)
			_, _ = crypto_rand.Read(b)
			offset = int(binary.BigEndian.Uint64(b) % uint64(totalUsable))
		}

		curr := begin
		for k := 0; k < offset; k++ {
			curr = nextIP(curr)
		}

		// 3. Search for the first available IP starting from offset
		for attempts := 0; attempts < totalUsable; attempts++ {
			ipStr := curr.String()

			// Skip gateway and network/broadcast
			if isNetworkOrBroadcast(curr, ipnet) || (s.Gateway != "" && ipStr == s.Gateway) {
				curr = i.nextWrapped(curr, begin, cutoff)
				continue
			}

			// Check if IP is taken using the SAME transaction
			var exists int64
			tx.Model(&IPAllocation{}).Where("subnet_id = ? AND ip = ?", s.ID, ipStr).Count(&exists)

			if exists == 0 {
				// FOUND! Create allocation under the lock.
				rec := IPAllocation{
					SubnetID: s.ID,
					IP:       ipStr,
					PortID:   portID,
				}
				if err := tx.Create(&rec).Error; err != nil {
					return fmt.Errorf("failed to persist allocation: %w", err)
				}
				allocatedIP = ipStr
				return nil
			}

			// Move to next candidate
			curr = i.nextWrapped(curr, begin, cutoff)
		}

		return fmt.Errorf("subnet %s is full", s.ID)
	})

	if err != nil {
		return "", err
	}
	return allocatedIP, nil
}

// nextWrapped moves to the next IP, wrapping around to begin if cutoff is reached.
func (i *IPAM) nextWrapped(curr, begin, cutoff net.IP) net.IP {
	if compareIP(curr, cutoff) >= 0 {
		return begin
	}
	return nextIP(curr)
}

func (i *IPAM) countRange(begin, end net.IP, ipnet *net.IPNet, gateway string) int {
	count := 0
	for ip := begin; compareIP(ip, end) <= 0; ip = nextIP(ip) {
		if !ipnet.Contains(ip) {
			break
		}
		count++
		if count > 2048 {
			return 2048
		} // Safety cap
	}
	return count
}

// Release frees an IP address.
func (i *IPAM) Release(ctx context.Context, subnetID, ip string) error {
	return i.db.WithContext(ctx).Where("subnet_id = ? AND ip = ?", subnetID, ip).Delete(&IPAllocation{}).Error
}

// --- IP Arithmetic Helpers ---

func firstUsableIP(n *net.IPNet, start string) net.IP {
	if start != "" {
		if ip := net.ParseIP(start); ip != nil {
			return ip
		}
	}
	ip := n.IP.Mask(n.Mask)
	return nextIP(ip)
}

func lastUsableIP(n *net.IPNet, end string) net.IP {
	if end != "" {
		if ip := net.ParseIP(end); ip != nil {
			return ip
		}
	}
	if v4 := n.IP.To4(); v4 != nil {
		bcast := make(net.IP, 4)
		for i := 0; i < 4; i++ {
			bcast[i] = n.IP.To4()[i] | ^n.Mask[i]
		}
		return prevIP(bcast)
	}
	return prevIP(net.IP(n.IP.Mask(n.Mask)))
}

func isNetworkOrBroadcast(ip net.IP, n *net.IPNet) bool {
	if !n.Contains(ip) {
		return false
	}
	if ip.Equal(n.IP.Mask(n.Mask)) {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		bcast := make(net.IP, 4)
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
	if len(b) < 16 {
		pad := make([]byte, 16-len(b))
		b = append(pad, b...)
	}
	return net.IP(b)
}

func prevIP(ip net.IP) net.IP {
	x := big.NewInt(0).SetBytes(ip.To16())
	x.Sub(x, big.NewInt(1))
	b := x.Bytes()
	if len(b) < 16 {
		pad := make([]byte, 16-len(b))
		b = append(pad, b...)
	}
	return net.IP(b)
}

func compareIP(a, b net.IP) int {
	return strings.Compare(string(a.To16()), string(b.To16()))
}
