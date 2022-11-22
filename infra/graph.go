package infra

import "fmt"

type GraphNode struct {
	Name       string
	Color      GraphNodeColor
	ParentNode []*GraphNode
	Async      bool
}

type GraphNodes []*GraphNode

func (nodes GraphNodes) Draw() string {
	graph := "digraph G {\n    node [shape = \"box\" style = \"filled,rounded\" fillcolor = \"gold\"]\n"
	for _, node := range nodes {
		if node.Color != "" {
			graph += fmt.Sprintf("    \"%s\" [fillcolor = \"%s\"]\n", node.Name, node.Color)
		}
		for _, parent := range node.ParentNode {
			graph += fmt.Sprintf("    \"%s\" -> \"%s\";\n", parent.Name, node.Name)
		}
	}
	graph += "}"
	return graph
}

type GraphNodeColor string

const (
	GraphNodeColorRed   GraphNodeColor = "red"
	GraphNodeColorBlue  GraphNodeColor = "darkturquoise"
	GraphNodeColorGreen GraphNodeColor = "chartreuse"
)
