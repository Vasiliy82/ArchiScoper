USE otel;

DROP TABLE IF EXISTS NodeDictionary;
DROP TABLE IF EXISTS NodeDictionaryMV;
DROP TABLE IF EXISTS TraceNodeMap;
DROP TABLE IF EXISTS TraceNodeMapMV;

CREATE TABLE NodeDictionary
(
    `NodeId` UInt32,
    `NodeUniqueName` String,
    `ServiceName` String,
    `Layer` String,
    `SubLayer` String,
    `CallCount` AggregateFunction(sum, UInt64),
    `P50Duration` AggregateFunction(quantiles(0.5), UInt64),
    `P90Duration` AggregateFunction(quantiles(0.9), UInt64),
    `P99Duration` AggregateFunction(quantiles(0.99), UInt64),
    `ErrorCount` AggregateFunction(sum, UInt64)
)
ENGINE = AggregatingMergeTree
ORDER BY NodeId;

CREATE MATERIALIZED VIEW NodeDictionaryMV TO NodeDictionary
AS SELECT
    cityHash64(CONCAT(SpanAttributes['function.name'], '.', SpanName)) AS NodeId,
    CONCAT(SpanAttributes['function.name'], '.', SpanName) AS NodeUniqueName,
    ServiceName,
    SpanAttributes['layer'] AS Layer,
    SpanAttributes['subLayer'] AS SubLayer,
    sumState(toUInt64(1)) AS CallCount,
    quantilesState(0.5)(Duration) AS P50Duration,
    quantilesState(0.9)(Duration) AS P90Duration,
    quantilesState(0.99)(Duration) AS P99Duration,
    sumState(toUInt64(if(StatusCode = 'Error', 1, 0))) AS ErrorCount
FROM otel_traces
GROUP BY
    NodeId,
    NodeUniqueName,
    ServiceName,
    Layer,
    SubLayer;

CREATE TABLE TraceNodeMap
(
    TraceId String, -- Уникальный идентификатор трассы
    SpanId String, -- Идентификатор спана
    NodeId UInt32 -- Сопоставленный `NodeId` из `NodeDictionary`
) ENGINE = MergeTree()
ORDER BY (TraceId, SpanId);


INSERT INTO TraceNodeMap
SELECT
    TraceId,
    SpanId,
    cityHash64(CONCAT(SpanAttributes['function.name'], '.', SpanName)) AS NodeId
FROM otel_traces;

CREATE MATERIALIZED VIEW TraceNodeMapMV TO TraceNodeMap AS
SELECT
    TraceId,
    SpanId,
    cityHash64(CONCAT(SpanAttributes['function.name'], '.', SpanName)) AS NodeId
FROM otel_traces;
