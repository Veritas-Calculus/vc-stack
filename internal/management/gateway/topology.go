package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// topologyHandler aggregates network and compute resources into a single topology graph.
// It calls underlying services with the same Authorization and X-Project-ID headers.
//
//nolint:gocyclo,gocognit // Complex topology aggregation logic
func (s *Service) topologyHandler(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	// Forward headers.
	auth := c.GetHeader("Authorization")
	projectHeader := c.GetHeader("X-Project-ID")

	type httpGetResult struct {
		body   []byte
		status int
		err    error
	}

	doGET := func(service string, path string, q string) httpGetResult {
		s.mu.RLock()
		proxy, ok := s.services[service]
		s.mu.RUnlock()
		if !ok {
			return httpGetResult{nil, http.StatusBadGateway, fmt.Errorf("service %s not found", service)}
		}
		url := fmt.Sprintf("%s%s", proxy.Target.String(), path)
		if q != "" {
			url = url + "?" + q
		}
		req, err := http.NewRequest("GET", url, http.NoBody)
		if err != nil {
			return httpGetResult{nil, http.StatusInternalServerError, err}
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		if projectHeader != "" {
			req.Header.Set("X-Project-ID", projectHeader)
		}
		// Propagate request ID for distributed tracing.
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}
		// Also pass tenant_id as header to services that rely on it.
		if tenantID != "" {
			req.Header.Set("X-Project-ID", tenantID)
		}
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		if err != nil {
			return httpGetResult{nil, http.StatusBadGateway, err}
		}
		defer func() { _ = resp.Body.Close() }()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			s.logger.Warn("failed to read response body", zap.Error(err))
		}
		return httpGetResult{b, resp.StatusCode, nil}
	}

	// Build query.
	q := ""
	if tenantID != "" {
		q = "tenant_id=" + tenantID
	}

	// Fetch resources in parallel (best-effort)
	// Use /api/v1/ prefix to match monolithic mode route registration.
	nets := doGET("network", "/api/v1/networks", q)
	subs := doGET("network", "/api/v1/subnets", q)
	rtrs := doGET("network", "/api/v1/routers", q)
	ports := doGET("network", "/api/v1/ports", q)
	insts := doGET("compute", "/api/v1/instances", q)

	// Minimal shapes for marshaling.
	var networks struct {
		Networks []map[string]interface{} `json:"networks"`
	}
	var subnets struct {
		Subnets []map[string]interface{} `json:"subnets"`
	}
	var routers []map[string]interface{}
	var routerWrap struct {
		Routers []map[string]interface{} `json:"routers"`
	}
	var portsObj struct {
		Ports []map[string]interface{} `json:"ports"`
	}
	var instancesObj struct {
		Instances []map[string]interface{} `json:"instances"`
	}

	// Decode forgivingly.
	if nets.status == http.StatusOK {
		if err := json.Unmarshal(nets.body, &networks); err != nil {
			s.logger.Warn("failed to unmarshal networks", zap.Error(err))
		}
	}
	if subs.status == http.StatusOK {
		// handle both array and object.
		if err := json.Unmarshal(subs.body, &subnets); err != nil {
			var arr []map[string]interface{}
			if err := json.Unmarshal(subs.body, &arr); err == nil {
				subnets.Subnets = arr
			}
		}
	}
	if rtrs.status == http.StatusOK {
		if err := json.Unmarshal(rtrs.body, &routerWrap); err == nil {
			routers = routerWrap.Routers
		} else {
			if err := json.Unmarshal(rtrs.body, &routers); err != nil {
				s.logger.Warn("failed to unmarshal routers", zap.Error(err))
			}
		}
	}
	if ports.status == http.StatusOK {
		if err := json.Unmarshal(ports.body, &portsObj); err != nil {
			s.logger.Warn("failed to unmarshal ports", zap.Error(err))
		}
	}
	if insts.status == http.StatusOK {
		if err := json.Unmarshal(insts.body, &instancesObj); err != nil {
			s.logger.Warn("failed to unmarshal instances", zap.Error(err))
		}
	}

	// Index helpers.
	get := func(m map[string]interface{}, k string) string {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	// Build nodes.
	nodes := make([]map[string]interface{}, 0, 64)
	edges := make([]map[string]string, 0, 128)

	// Networks.
	for _, n := range networks.Networks {
		nodes = append(nodes, map[string]interface{}{
			"id":               "net-" + get(n, "id"),
			"resource_id":      get(n, "id"),
			"type":             "network",
			"name":             get(n, "name"),
			"cidr":             get(n, "cidr"),
			"external":         n["external"],
			"network_type":     n["network_type"],
			"segmentation_id":  n["segmentation_id"],
			"shared":           n["shared"],
			"physical_network": n["physical_network"],
			"mtu":              n["mtu"],
		})
	}

	// Subnets + edges to networks.
	for _, sObj := range subnets.Subnets {
		sid := get(sObj, "id")
		nid := get(sObj, "network_id")
		nodes = append(nodes, map[string]interface{}{
			"id":          "subnet-" + sid,
			"resource_id": sid,
			"type":        "subnet",
			"name":        get(sObj, "name"),
			"cidr":        get(sObj, "cidr"),
			"gateway":     get(sObj, "gateway"),
			"network_id":  nid,
		})
		if nid != "" {
			edges = append(edges, map[string]string{
				"source": "subnet-" + sid,
				"target": "net-" + nid,
				"type":   "l2",
			})
		}
	}

	// Routers.
	for _, r := range routers {
		rid := get(r, "id")
		nodes = append(nodes, map[string]interface{}{
			"id":                          "router-" + rid,
			"resource_id":                 rid,
			"type":                        "router",
			"name":                        get(r, "name"),
			"enable_snat":                 r["enable_snat"],
			"external_gateway_network_id": r["external_gateway_network_id"],
			"external_gateway_ip":         r["external_gateway_ip"],
		})
		// Router gateway to external network.
		extNet := get(r, "external_gateway_network_id")
		if extNet != "" {
			edges = append(edges, map[string]string{
				"source": "router-" + rid,
				"target": "net-" + extNet,
				"type":   "l3-gateway",
			})
		}
	}

	// Router interfaces: need to query per-router.
	for _, r := range routers {
		rid := get(r, "id")
		path := fmt.Sprintf("/api/v1/routers/%s/interfaces", rid)
		ris := doGET("network", path, "")
		if ris.status != http.StatusOK || len(ris.body) == 0 {
			continue
		}
		var ifaces []map[string]interface{}
		if err := json.Unmarshal(ris.body, &ifaces); err == nil {
			connected := make([]string, 0, len(ifaces))
			for _, iface := range ifaces {
				subID := get(iface, "subnet_id")
				if subID != "" {
					edges = append(edges, map[string]string{
						"source": "router-" + rid,
						"target": "subnet-" + subID,
						"type":   "l3",
					})
					connected = append(connected, subID)
				}
			}
			// annotate router node with interface subnet ids.
			for i := range nodes {
				if nodes[i]["id"] == "router-"+rid {
					nodes[i]["interfaces"] = connected
					break
				}
			}
		}
	}

	// Instances.
	for _, inst := range instancesObj.Instances {
		iid := get(inst, "id")
		// derive primary IP from ports (first fixed_ips entry)
		var primaryIP string
		for _, p := range portsObj.Ports {
			if get(p, "device_id") == iid {
				if v, ok := p["fixed_ips"]; ok && v != nil {
					if arr, ok2 := v.([]interface{}); ok2 && len(arr) > 0 {
						if ipm, ok3 := arr[0].(map[string]interface{}); ok3 {
							if ipStr, ok4 := ipm["ip"].(string); ok4 {
								primaryIP = ipStr
							}
						}
					}
				}
				if primaryIP != "" {
					break
				}
			}
		}
		nodes = append(nodes, map[string]interface{}{
			"id":          "instance-" + iid,
			"resource_id": iid,
			"type":        "instance",
			"name":        get(inst, "name"),
			"state":       get(inst, "status"),
			"ip":          primaryIP,
		})
	}

	// Ports: connect instances to subnets (or networks)
	for _, p := range portsObj.Ports {
		devID := get(p, "device_id")
		if devID == "" {
			continue
		}
		// prefer subnet_id from port; if missing, connect to network.
		sid := get(p, "subnet_id")
		if sid != "" {
			edges = append(edges, map[string]string{
				"source": "instance-" + devID,
				"target": "subnet-" + sid,
				"type":   "attachment",
			})
			continue
		}
		nid := get(p, "network_id")
		if nid != "" {
			edges = append(edges, map[string]string{
				"source": "instance-" + devID,
				"target": "net-" + nid,
				"type":   "attachment",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"edges": edges,
		"meta":  gin.H{"generated_at": time.Now()},
	})
}
