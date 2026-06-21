package indexer

import "strings"

func (b *graphBuilder) reconcilePlaceholders() {
	for id, edge := range b.edges {
		const prefix = "placeholder:package:"
		if edge.Kind != "imports" || !strings.HasPrefix(edge.To, prefix) {
			continue
		}
		realID := "package:" + strings.TrimPrefix(edge.To, prefix)
		if !b.hasNode(realID) {
			continue
		}
		edge.To = realID
		edge.ID = "edge:imports:" + edge.From + "->" + realID
		edge.Synthetic = false
		edge.Complete = true
		edge.Reason = "placeholder_reconciled_to_real_package"
		delete(b.edges, id)
		b.edges[edge.ID] = edge
		delete(b.nodes, "placeholder:"+realID)
	}
}
