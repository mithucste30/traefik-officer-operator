package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TargetReference references a target resource (Ingress or IngressRoute)
type TargetReference struct {
	// Kind of the target resource (Ingress or IngressRoute)
	// +kubebuilder:validation:Enum=Ingress;IngressRoute
	// +kubebuilder:default=Ingress
	Kind string `json:"kind"`

	// Name of the target resource
	Name string `json:"name"`

	// Namespace of the target resource.
	// Defaults to the namespace of the UrlPerformance resource.
	Namespace string `json:"namespace,omitempty"`
}

// URLPattern defines a custom regex pattern for URL normalization
type URLPattern struct {
	// Regex pattern to match URLs
	Pattern string `json:"pattern"`

	// Replacement string for matched URLs
	Replacement string `json:"replacement"`
}

// UrlPerformanceSpec defines the desired state of UrlPerformance
type UrlPerformanceSpec struct {
	// TargetRef references the Ingress or IngressRoute to monitor
	TargetRef TargetReference `json:"targetRef"`

	// WhitelistPathsRegex is a list of regex patterns.
	// Only paths matching these patterns will be monitored for the target ingress.
	// If empty, all paths are monitored (unless ignored).
	// +optional
	// +kubebuilder:default=[]
	WhitelistPathsRegex []string `json:"whitelistPathsRegex,omitempty"`

	// IgnoredPathsRegex is a list of regex patterns.
	// Paths matching these patterns will always be ignored for the target ingress.
	// +optional
	// +kubebuilder:default=[]
	IgnoredPathsRegex []string `json:"ignoredPathsRegex,omitempty"`

	// MergePathsWithExtensions is a list of path prefixes.
	// Paths under these prefixes will be merged (query parameters and path parameters replaced).
	// +optional
	// +kubebuilder:default=[]
	MergePathsWithExtensions []string `json:"mergePathsWithExtensions,omitempty"`

	// URLPatterns defines custom regex patterns for URL normalization.
	// +optional
	URLPatterns []URLPattern `json:"urlPatterns,omitempty"`

	// CollectNTop specifies the number of top URL paths (by latency) to collect detailed metrics for.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	// +kubebuilder:default=20
	// +default=20
	CollectNTop int `json:"collectNTop,omitempty"`

	// Enabled controls whether monitoring is active for this resource.
	// +optional
	// +kubebuilder:default=true
	// +default=true
	Enabled bool `json:"enabled,omitempty"`
}

// ConditionType represents a condition type
type ConditionType string

const (
	// ConditionReady indicates the resource is ready
	ConditionReady ConditionType = "Ready"
	// ConditionTargetExists indicates the target resource exists
	ConditionTargetExists ConditionType = "TargetExists"
	// ConditionConfigGenerated indicates configuration has been generated
	ConditionConfigGenerated ConditionType = "ConfigGenerated"
)

// Condition defines an observation of a UrlPerformance's state
type Condition struct {
	// Type of the condition
	Type ConditionType `json:"type"`

	// Status of the condition (True, False, Unknown)
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status string `json:"status"`

	// LastTransitionTime is the last time the condition transitioned
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason indicates the reason for the condition's last transition
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message provides a human-readable explanation of the condition
	// +optional
	Message string `json:"message,omitempty"`
}

// Phase represents the current state of UrlPerformance
type Phase string

const (
	// PhasePending indicates the resource is pending initialization
	PhasePending Phase = "Pending"
	// PhaseActive indicates the resource is actively monitoring
	PhaseActive Phase = "Active"
	// PhaseError indicates the resource has encountered an error
	PhaseError Phase = "Error"
	// PhaseDisabled indicates the resource is disabled
	PhaseDisabled Phase = "Disabled"
)

// UrlPerformanceStatus defines the observed state of UrlPerformance
type UrlPerformanceStatus struct {
	// Phase indicates the current state of the UrlPerformance resource
	// +kubebuilder:validation:Enum=Pending;Active;Error;Disabled
	// +kubebuilder:default=Pending
	Phase Phase `json:"phase,omitempty"`

	// Conditions represents the latest available observations of the UrlPerformance state
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`

	// MonitoredPaths is the count of unique paths currently being monitored for this resource
	// +optional
	MonitoredPaths int32 `json:"monitoredPaths,omitempty"`

	// LastScrapeTime is the timestamp when metrics were last collected
	// +optional
	LastScrapeTime *metav1.Time `json:"lastScrapeTime,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=urlperf
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Target Kind",type=string,JSONPath=`.spec.targetRef.kind`,description="The kind of target resource"
// +kubebuilder:printcolumn:name="Target Name",type=string,JSONPath=`.spec.targetRef.name`,description="The name of the target resource"
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.targetRef.namespace`,description="The namespace of the target resource"
// +kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=`.spec.enabled`,description="Whether monitoring is enabled"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="The current phase"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="The age of the resource"

// UrlPerformance is the Schema for the urlperformances API
type UrlPerformance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UrlPerformanceSpec   `json:"spec,omitempty"`
	Status UrlPerformanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UrlPerformanceList contains a list of UrlPerformance
type UrlPerformanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UrlPerformance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UrlPerformance{}, &UrlPerformanceList{})
}
