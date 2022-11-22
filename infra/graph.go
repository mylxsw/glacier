package infra

import "fmt"

type GraphvizNode struct {
	Name       string
	Style      GraphvizNodeStyle
	ParentNode []*GraphvizNode
	Async      bool
	Type       GraphvizNodeType
}

type GraphvizNodes []*GraphvizNode

func (nodes GraphvizNodes) Draw() string {
	graph := `digraph G {
    node [shape = "box" style = "filled,rounded" fillcolor = "gold"]
`
	clusters := make(map[string][]string)
	var cluster []string
	var clusterName string
	for _, node := range nodes {
		if node.Type == GraphvizNodeTypeClusterStart {
			cluster = make([]string, 0)
			clusterName = node.Name
			continue
		}

		if node.Type == GraphvizNodeTypeClusterEnd {
			clusters[clusterName] = cluster
			clusterName = ""
			cluster = nil
			continue
		}

		if cluster != nil {
			cluster = append(cluster, node.Name)
		}

		if node.Style != "" {
			graph += fmt.Sprintf("    \"%s\" %s\n", node.Name, node.Style)
		}
		for _, parent := range node.ParentNode {
			graph += fmt.Sprintf("    \"%s\" -> \"%s\";\n", parent.Name, node.Name)
		}
	}

	for name, cluster := range clusters {
		graph += fmt.Sprintf(`    subgraph cluster_%s {
        label = "%s"
        style = "rounded,dashed,filled"
        color = "deepskyblue"
        fillcolor = "aliceblue" 
`, name, name)

		for _, node := range cluster {
			graph += fmt.Sprintf("        \"%s\"\n", node)
		}

		graph += "    }\n"
	}

	graph += "}"
	return graph
}

type GraphvizNodeStyle string

const (
	GraphvizNodeStyleError     GraphvizNodeStyle = `[fillcolor = "red"]`
	GraphvizNodeStyleHook      GraphvizNodeStyle = `[fillcolor = "darkturquoise"]`
	GraphvizNodeStyleImportant GraphvizNodeStyle = `[fillcolor = "chartreuse"]`
)

type GraphvizNodeType string

const (
	GraphvizNodeTypeNode         GraphvizNodeType = "node"
	GraphvizNodeTypeClusterStart GraphvizNodeType = "cluster_start"
	GraphvizNodeTypeClusterEnd   GraphvizNodeType = "cluster_end"
)
