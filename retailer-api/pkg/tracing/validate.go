package tracing

import "fmt"

type Layer string
type SubLayer string

const (
	LayerApplication    Layer = "application"
	LayerPresentation   Layer = "presentation"
	LayerInfrastructure Layer = "infrastructure"
	LayerIntegration    Layer = "integration"
)

const (
	SubLayerUseCase   SubLayer = "usecase"
	SubLayerService   SubLayer = "service"
	SubLayerValidator SubLayer = "validator"

	SubLayerDatabase   SubLayer = "database"
	SubLayerBroker     SubLayer = "broker"
	SubLayerCache      SubLayer = "cache"
	SubLayerFilesystem SubLayer = "filesystem"
	SubLayerAuth       SubLayer = "auth"
	SubLayerMetrics    SubLayer = "metrics"

	SubLayerHTTP       SubLayer = "http"
	SubLayerGRPC       SubLayer = "grpc"
	SubLayerWebhook    SubLayer = "webhook"
	SubLayerThirdParty SubLayer = "thirdparty"
)

// Определяем допустимые комбинации Layer -> SubLayer
var validSubLayers = map[Layer][]SubLayer{
	LayerApplication:    {SubLayerUseCase, SubLayerService, SubLayerValidator},
	LayerPresentation:   {SubLayerHTTP, SubLayerGRPC},
	LayerInfrastructure: {SubLayerDatabase, SubLayerBroker, SubLayerCache, SubLayerFilesystem, SubLayerAuth, SubLayerMetrics},
	LayerIntegration:    {SubLayerHTTP, SubLayerGRPC, SubLayerWebhook, SubLayerThirdParty},
}

// Функция проверки валидности слоев
func validateLayerSubLayer(layer Layer, subLayer SubLayer) error {
	validSubLayersForLayer, exists := validSubLayers[layer]
	if !exists {
		return fmt.Errorf("unknown layer: %s", layer)
	}
	for _, validSubLayer := range validSubLayersForLayer {
		if validSubLayer == subLayer {
			return nil // Всё корректно
		}
	}
	return fmt.Errorf("invalid sublayer %s for layer %s", subLayer, layer)
}
