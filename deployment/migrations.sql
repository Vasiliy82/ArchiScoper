USE otel;

DROP TABLE IF EXISTS NodeDictionary;
DROP TABLE IF EXISTS NodeDictionaryMV;
DROP TABLE IF EXISTS TraceNodeMap;
DROP TABLE IF EXISTS TraceNodeMapMV;

CREATE TABLE NodeDictionary
(
    NodeId UInt32, -- Уникальный короткий идентификатор узла
    NodeUniqueName String, -- Полное имя узла (SpanAttributes['function.name'] + SpanName)
    ServiceName String, -- Имя сервиса (для субграфов и минимизации)
    Layer String, -- Имя слоя (для субграфов и минимизации)
    SubLayer String, -- Имя подслоя (для субграфов и минимизации)
    CallCount UInt64, -- Количество вызовов
    P50Duration UInt64, -- 50-й перцентиль времени выполнения
    P90Duration UInt64, -- 90-й перцентиль
    P99Duration UInt64, -- 99-й перцентиль
    ErrorCount UInt64 -- Количество ошибок
) ENGINE = ReplacingMergeTree()
ORDER BY NodeId;


INSERT INTO NodeDictionary
SELECT
    cityHash64(CONCAT(SpanAttributes['function.name'], '.', SpanName)) AS NodeId,
    CONCAT(SpanAttributes['function.name'], '.', SpanName) AS NodeUniqueName,
    ServiceName,
    SpanAttributes['layer'] AS Layer,
    SpanAttributes['subLayer'] AS SubLayer,
    count() AS CallCount,
    quantile(0.50)(Duration) AS P50Duration,
    quantile(0.90)(Duration) AS P90Duration,
    quantile(0.99)(Duration) AS P99Duration,
    countIf(StatusCode = 'Error') AS ErrorCount
FROM otel_traces
GROUP BY NodeUniqueName, ServiceName, SpanAttributes['layer'],
    SpanAttributes['subLayer'];


CREATE MATERIALIZED VIEW NodeDictionaryMV TO NodeDictionary AS
SELECT
    cityHash64(CONCAT(SpanAttributes['function.name'], '.', SpanName)) AS NodeId,
    CONCAT(SpanAttributes['function.name'], '.', SpanName) AS NodeUniqueName,
    ServiceName,
    SpanAttributes['layer'] AS Layer,
    SpanAttributes['subLayer'] AS SubLayer,
    count() AS CallCount,
    quantile(0.50)(Duration) AS P50Duration,
    quantile(0.90)(Duration) AS P90Duration,
    quantile(0.99)(Duration) AS P99Duration,
    countIf(StatusCode = 'Error') AS ErrorCount
FROM otel_traces
GROUP BY NodeUniqueName, ServiceName, SpanAttributes['layer'],
    SpanAttributes['subLayer'];

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
