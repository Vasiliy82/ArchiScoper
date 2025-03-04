package main

import (
	"database/sql"
	"fmt"
	"log"

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
		SELECT NodeId, NodeUniqueName, ServiceName, Layer, SubLayer, CallCount, P50Duration, P90Duration, P99Duration, ErrorCount
		FROM NodeDictionary
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
						graph.AddEdge(linkedNodeID, currentNodeID, duration, isError, "async", protocol)
					}
				}
			}
		}
	}

	// **Шаг 1: Определяем узлы, которые являются точками входа в микросервис**
	entryPoints := make(map[string]Node) // serviceName -> aggregated Node

	for _, node := range graph.Nodes {
		isEntryPoint := true

		// Проверяем, есть ли у узла родитель в этом же микросервисе
		for src, targets := range graph.Edges {
			if _, exists := targets[node.NodeID]; exists {
				if graph.Nodes[src].ServiceName == node.ServiceName {
					isEntryPoint = false
					break
				}
			}
		}

		if isEntryPoint {
			service := node.ServiceName
			if existing, ok := entryPoints[service]; ok {
				// Агрегируем метрики
				existing.CallCount = max(existing.CallCount, node.CallCount)
				existing.ErrorCount += node.ErrorCount
				entryPoints[service] = existing
			} else {
				// Создаём новый агрегированный узел
				entryPoints[service] = Node{
					NodeID:      node.NodeID,
					NodeName:    service, // Теперь узел = микросервису
					ServiceName: service,
					Layer:       node.Layer,
					SubLayer:    node.SubLayer,
					CallCount:   node.CallCount,
					ErrorCount:  node.ErrorCount,
				}
			}
		}
	}

	// **Шаг 2: Перепривязываем связи между микросервисами**
	collapsedEdges := make(map[string]map[string]EdgeMetrics) // Исходный сервис → Целевой сервис → Метрики

	for src, targets := range graph.Edges {
		srcService := graph.Nodes[src].ServiceName
		for dst, metrics := range targets {
			dstService := graph.Nodes[dst].ServiceName

			// Перепривязываем связи только между микросервисами
			if srcService == dstService {
				continue
			}

			if collapsedEdges[srcService] == nil {
				collapsedEdges[srcService] = make(map[string]EdgeMetrics)
			}
			aggMetrics := collapsedEdges[srcService][dstService]

			// Агрегируем связи между микросервисами
			aggMetrics.Count += metrics.Count
			aggMetrics.TotalTime += metrics.TotalTime
			aggMetrics.ErrorCount += metrics.ErrorCount
			aggMetrics.Type = metrics.Type
			aggMetrics.Protocol = metrics.Protocol

			collapsedEdges[srcService][dstService] = aggMetrics
		}
	}

	// **Шаг 3: Вывод нового графа**
	fmt.Println("digraph G {")

	// **Вывод узлов (по одному на микросервис)**
	for _, node := range entryPoints {
		fmt.Printf("  \"%s\" [label=\"%s\\nCalls: %d\\nErrors: %d\", shape=box];\n",
			node.ServiceName, node.NodeName, node.CallCount, node.ErrorCount)
	}

	// **Вывод рёбер (связи между микросервисами)**
	for src, targets := range collapsedEdges {
		for dst, metrics := range targets {
			color := "black"
			style := "solid"
			if metrics.ErrorCount > 0 {
				color = "red"
			}
			if metrics.Type == "async" {
				style = "dashed"
			}
			protocolLabel := ""
			if metrics.Protocol != "" {
				protocolLabel = fmt.Sprintf(", label=\"%s\"", metrics.Protocol)
			}
			fmt.Printf("  \"%s\" -> \"%s\" [label=\"%s, %d calls, avg %d ms, %d errors\"%s, color=%s, style=%s];\n",
				src, dst, metrics.Type, metrics.Count, metrics.TotalTime/metrics.Count/1e6, metrics.ErrorCount, protocolLabel, color, style)
		}
	}

	fmt.Println("}")

}

// Функция для max()
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
