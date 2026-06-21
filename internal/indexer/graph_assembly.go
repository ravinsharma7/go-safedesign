package indexer

import (
	"strings"

	"go-safedesign/internal/core"
)

func (b *graphBuilder) reconcilePlaceholders() {
	for id, edge := range b.edges {
		if edge.Kind != core.EdgeKindImports || !strings.HasPrefix(edge.To, core.IDPrefixPlaceholderPackage) {
			continue
		}
		realID := core.PackageID(strings.TrimPrefix(edge.To, core.IDPrefixPlaceholderPackage))
		if !b.hasNode(realID) {
			continue
		}
		edge.To = realID
		edge.ID = core.EdgeID(core.EdgeKindImports, edge.From, realID)
		edge.Synthetic = false
		edge.Complete = true
		edge.Reason = "placeholder_reconciled_to_real_package"
		delete(b.edges, id)
		b.edges[edge.ID] = edge
		delete(b.nodes, core.IDPrefixPlaceholder+realID)
	}
}
