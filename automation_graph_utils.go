package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// RewriteAutomationNodeIds replaces every node ID in the graph with a fresh
// random value, updating edge references to match. This prevents ID collisions
// when cloning or importing automations.
func RewriteAutomationNodeIds(g *AutomationGraph) {
	idMap := make(map[string]string, len(g.Nodes))
	for i, n := range g.Nodes {
		newId := NewAutomationNodeId()
		idMap[n.Id] = newId
		g.Nodes[i].Id = newId
	}
	for i, e := range g.Edges {
		if mapped, ok := idMap[e.From]; ok {
			g.Edges[i].From = mapped
		}
		if mapped, ok := idMap[e.To]; ok {
			g.Edges[i].To = mapped
		}
	}
}

// NewAutomationNodeId generates a random node identifier of the form "n-<hex>".
func NewAutomationNodeId() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("n-%d", time.Now().UnixNano())
	}
	return "n-" + hex.EncodeToString(b)
}
