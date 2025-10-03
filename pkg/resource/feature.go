package resource

type FeatureName string

const FeaturePrefix = "features"

type FeatureState string

const FeatureStateActive FeatureState = "active"
const FeatureStateDisabled FeatureState = "disabled"

type Feature struct {
	Name        FeatureName  `json:"name"`
	State       FeatureState `json:"state"`
	Description string       `json:"description"`
}
