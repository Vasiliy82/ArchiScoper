package tracing

import (
	"fmt"
	"runtime"
)

// Получение имени вызывающей функции
func getCallerFunctionName() string {
	pc, _, _, ok := runtime.Caller(3) // 3 уровня вверх
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

// Формирование имени спана по шаблону
func generateSpanName(operation string, layer Layer, subLayer SubLayer) string {
	// Базовое имя
	spanName := operation

	// Добавляем слой, если это необходимо
	if layer == LayerApplication {
		spanName = fmt.Sprintf("Business %s", operation)
	} else if layer == LayerInfrastructure && subLayer == SubLayerDatabase {
		spanName = fmt.Sprintf("SQL %s", operation)
	} else if layer == LayerIntegration && subLayer == SubLayerThirdParty {
		spanName = fmt.Sprintf("External API %s", operation)
	} else if layer == LayerInfrastructure && subLayer == SubLayerBroker {
		spanName = fmt.Sprintf("Kafka %s", operation)
	}

	return spanName
}
