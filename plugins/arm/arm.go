package plugins_arm

// Struct representing the final deployment metadata plan stored by ARM
type deployment struct {
	Name       string               `json:"name"`
	Properties deploymentProperties `json:"properties"`
}

// Struct that contains a list of resources created as part of a template
type deploymentProperties struct {
	OutputResources []outputResource `json:"outputResources"`
}

// Struct that contains each resource created by a template
type outputResource struct {
	ID string `json:"id"`
}
