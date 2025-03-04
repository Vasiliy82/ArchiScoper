package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// Node представляет узел графа
type Node struct {
	NodeID      uint32
	NodeName    string
	ServiceName string
	Layer       string
	SubLayer    string
	CallCount   uint64
	P50Duration uint64
	P90Duration uint64
	P99Duration uint64
	ErrorCount  uint64
}

// EdgeMetrics агрегирует количество вызовов, время выполнения и ошибки
type EdgeMetrics struct {
	Protocol   string
	Type       string
	Count      int64
	ErrorCount int64
	TotalTime  int64
}

// Link представляет асинхронную связь между спанами
type Link struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	Attributes map[string]string `json:"attributes"`
}

// Graph содержит узлы и связи между ними
type Graph struct {
	Nodes     map[uint32]Node
	Edges     map[uint32]map[uint32]EdgeMetrics
	Subgraphs map[string][]uint32 // Группировка узлов по ServiceName
}

func NewGraph() *Graph {
	return &Graph{
		Nodes:     make(map[uint32]Node),
		Edges:     make(map[uint32]map[uint32]EdgeMetrics),
		Subgraphs: make(map[string][]uint32),
	}
}

func (g *Graph) AddNode(node Node) {
	g.Nodes[node.NodeID] = node
	g.Subgraphs[node.ServiceName] = append(g.Subgraphs[node.ServiceName], node.NodeID)
}

func (g *Graph) AddEdge(from, to uint32, duration int64, isError bool, callType, protocol string) {
	if g.Edges[from] == nil {
		g.Edges[from] = make(map[uint32]EdgeMetrics)
	}
	metrics := g.Edges[from][to]
	metrics.Count++
	metrics.TotalTime += duration
	if isError {
		metrics.ErrorCount++
	}
	if callType != "" {
		metrics.Type = callType
	}
	if protocol != "" {
		metrics.Protocol = protocol
	}
	g.Edges[from][to] = metrics
}

func main() {
	conn, err := sql.Open("clickhouse", "tcp://localhost:9000/otel?username=otel&password=otel_passwd")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// **1. Загружаем все узлы (NodeDictionary)**
	rows, err := conn.Query(`
		SELECT 
			NodeId, 
			NodeUniqueName, 
			ServiceName, 
			Layer, 
			SubLayer, 
			sumMerge(CallCount) AS CallCount,
			toUInt64(quantilesMerge(0.50)(P50Duration)[1]) AS P50Duration,
			toUInt64(quantilesMerge(0.90)(P90Duration)[1]) AS P90Duration,
			toUInt64(quantilesMerge(0.99)(P99Duration)[1]) AS P99Duration,
			sumMerge(ErrorCount) AS ErrorCount
		FROM NodeDictionary FINAL
		GROUP BY NodeId, NodeUniqueName, ServiceName, Layer, SubLayer;
`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	graph := NewGraph()

	for rows.Next() {
		var node Node
		err := rows.Scan(&node.NodeID, &node.NodeName, &node.ServiceName, &node.Layer, &node.SubLayer, &node.CallCount, &node.P50Duration, &node.P90Duration, &node.P99Duration, &node.ErrorCount)
		if err != nil {
			log.Fatal(err)
		}
		graph.AddNode(node)
	}

	// **2. Загружаем маппинг (TraceId, SpanId, NodeId)**
	traceNodeMap := make(map[string]map[string]uint32) // TraceId -> SpanId -> NodeId

	rows, err = conn.Query(`SELECT TraceId, SpanId, NodeId FROM TraceNodeMap`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var traceID, spanID string
		var nodeID uint32
		err := rows.Scan(&traceID, &spanID, &nodeID)
		if err != nil {
			log.Fatal(err)
		}
		if traceNodeMap[traceID] == nil {
			traceNodeMap[traceID] = make(map[string]uint32)
		}
		traceNodeMap[traceID][spanID] = nodeID
	}

	// fmt.Println("Loaded TraceNodeMap:")
	// for tID, spans := range traceNodeMap {
	// 	fmt.Printf("TraceID: %s\n", tID)
	// 	for sID, nID := range spans {
	// 		fmt.Printf("  SpanID: %s -> NodeID: %d\n", sID, nID)
	// 	}
	// }

	// **3. Сканируем трейсы и строим связи**
	rows, err = conn.Query(`
		SELECT TraceId, SpanId, ParentSpanId, Duration, StatusCode, Links		       
		FROM otel_traces
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var traceID, spanID, parentID, statusCode string
		var duration int64
		var rawLinks []map[string]interface{}
		var links []Link

		err := rows.Scan(&traceID, &spanID, &parentID, &duration, &statusCode, &rawLinks)
		if err != nil {
			log.Fatal(err)
		}

		isError := (statusCode == "Error")

		// **Обрабатываем Links (асинхронные вызовы)**
		for _, raw := range rawLinks {
			link := Link{}

			if linkTraceID, ok := raw["TraceId"].(string); ok {
				link.TraceID = linkTraceID
			}
			if linkSpanID, ok := raw["SpanId"].(string); ok {
				link.SpanID = linkSpanID
			}
			if attributes, ok := raw["Attributes"].(map[string]string); ok {
				link.Attributes = attributes
			}

			links = append(links, link)
		}

		// **Добавляем синхронные вызовы**
		currentNodeID, existsCurrent := traceNodeMap[traceID][spanID]
		parentNodeID, existsParent := traceNodeMap[traceID][parentID]

		if existsCurrent && existsParent {
			graph.AddEdge(parentNodeID, currentNodeID, duration, isError, "sync", "")
		}

		// **Добавляем асинхронные вызовы**
		for _, link := range links {
			if linkedTraceMap, exists := traceNodeMap[link.TraceID]; exists {
				if linkedNodeID, found := linkedTraceMap[link.SpanID]; found {
					if existsCurrent {
						protocol := link.Attributes["link.protocol"]
						typ := link.Attributes["link.type"]
						graph.AddEdge(linkedNodeID, currentNodeID, duration, isError, typ, protocol)
					}
				}
			}
		}
	}

	// **4. Выводим граф в DOT-формате**
	fmt.Println("digraph G {")

	// **Рисуем субграфы (ServiceName)**
	for serviceName, nodes := range graph.Subgraphs {
		fmt.Printf("  subgraph cluster_%s {\n", strings.ReplaceAll(serviceName, "-", "_"))
		fmt.Printf("    label=\"%s\";\n", serviceName)
		for _, nodeID := range nodes {
			node := graph.Nodes[nodeID]
			fmt.Printf("    \"%d\" [label=\"%s\\nCalls: %d\\nP50: %dms\\nP90: %dms\\nP99: %dms\\nErrors: %d\", shape=box];\n",
				nodeID, extractShortName(node.NodeName), node.CallCount, node.P50Duration/1e6, node.P90Duration/1e6, node.P99Duration/1e6, node.ErrorCount)
		}
		fmt.Println("  }")
	}

	// **Рисуем рёбра**
	for src, targets := range graph.Edges {
		for dst, metrics := range targets {
			color := "black"
			style := "solid"
			if metrics.ErrorCount > 0 {
				color = "red"
			}
			if metrics.Type == "saga" {
				color = "blue"
			}
			if metrics.Type == "async" {
				style = "dashed" // Делаем пунктирную линию для асинхронных вызовов
			}
			protocolLabel := ""
			if metrics.Protocol != "" {
				protocolLabel = fmt.Sprintf(", label=\"%s\"", metrics.Protocol)
			}
			fmt.Printf("  \"%d\" -> \"%d\" [label=\"%s, %d calls, avg %d ms, %d errors\"%s, color=%s, style=%s];\n",
				src, dst, metrics.Type, metrics.Count, metrics.TotalTime/metrics.Count/1e6, metrics.ErrorCount, protocolLabel, color, style)
		}
	}

	fmt.Println("}")
}

func extractShortName(fullName string) string {
	lastDot := strings.LastIndex(fullName, ".")
	if lastDot == -1 {
		return fullName // Если точка не найдена, возвращаем исходную строку
	}
	return fullName[lastDot+1:]
}
