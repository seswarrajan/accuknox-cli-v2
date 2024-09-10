package policy

import (
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Global Alias: stores the label of an endpoint
type LabelMap = map[string]string

/* KubeArmor system policy */
// KubeArmorPolicy Structure
type KubeArmorPolicy struct {
	APIVersion string            `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string            `json:"kind,omitempty" yaml:"kind,omitempty"`
	Metadata   metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Spec KnoxSystemSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Spec HostSecuritySpec `json:"spec" yaml:"spec"`
}

// CustomUnmarshalJSON implements custom JSON unmarshalling due to
// inconsistencies in mapping between files stored in /opt/... and
// type defined in metav1.ObjectMeta
func (k *KubeArmorPolicy) CustomUnmarshalJSON(data []byte) error {
	type Alias KubeArmorPolicy
	aux := &struct {
		Metadata struct {
			PolicyName string `json:"policyName"`
		} `json:"metadata"`
		*Alias
	}{
		Alias: (*Alias)(k),
	}

	if err := jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(data, &aux); err != nil {
		return err
	}

	k.Metadata.Name = aux.Metadata.PolicyName
	return nil
}

// New method for generating optimized YAML for markdown display
type OptimizedMatch struct {
	Path     string `yaml:"path,omitempty"`
	Dir      string `yaml:"dir,omitempty"`
	Pattern  string `yaml:"pattern,omitempty"`
	Severity int    `yaml:"severity,omitempty"`
	Action   string `yaml:"action,omitempty"`
}

type OptimizedFile struct {
	MatchPaths       []OptimizedMatch `yaml:"matchPaths,omitempty"`
	MatchDirectories []OptimizedMatch `yaml:"matchDirectories,omitempty"`
	MatchPatterns    []OptimizedMatch `yaml:"matchPatterns,omitempty"`
	Severity         int              `yaml:"severity,omitempty"`
	Action           string           `yaml:"action,omitempty"`
}

type OptimizedSpec struct {
	NodeSelector yaml.MapSlice `yaml:"nodeSelector,omitempty"`
	File         OptimizedFile `yaml:"file,omitempty"`
	Severity     int           `yaml:"severity,omitempty"`
	Tags         []string      `yaml:"tags,omitempty"`
	Message      string        `yaml:"message,omitempty"`
	Action       string        `yaml:"action,omitempty"`
}

func (k KubeArmorPolicy) OptimizedYAML() (string, error) {

	optimized := struct {
		APIVersion string            `yaml:"apiVersion,omitempty"`
		Kind       string            `yaml:"kind,omitempty"`
		Metadata   metav1.ObjectMeta `yaml:"metadata,omitempty"`
		Spec       OptimizedSpec     `yaml:"spec"`
	}{
		APIVersion: k.APIVersion,
		Kind:       k.Kind,
		Metadata:   k.Metadata,
		Spec: OptimizedSpec{
			NodeSelector: yaml.MapSlice{
				{Key: "matchLabels", Value: k.Spec.NodeSelector.MatchLabels},
				{Key: "identities", Value: k.Spec.NodeSelector.Identities},
			},
			File: OptimizedFile{
				MatchPaths:       optimizeFilePaths(k.Spec.File.MatchPaths),
				MatchDirectories: optimizeFileDirectories(k.Spec.File.MatchDirectories),
				MatchPatterns:    optimizeFilePatterns(k.Spec.File.MatchPatterns),
				Severity:         k.Spec.File.Severity,
				Action:           k.Spec.File.Action,
			},
			Severity: k.Spec.Severity,
			Tags:     k.Spec.Tags,
			Message:  k.Spec.Message,
			Action:   k.Spec.Action,
		},
	}

	yamlBytes, err := yaml.Marshal(optimized)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}

func optimizeFilePaths(paths []FilePathType) []OptimizedMatch {
	optimized := make([]OptimizedMatch, len(paths))
	for i, p := range paths {
		optimized[i] = OptimizedMatch{
			Path:     p.Path,
			Severity: p.Severity,
			Action:   p.Action,
		}
	}
	return optimized
}

func optimizeFileDirectories(dirs []FileDirectoryType) []OptimizedMatch {
	optimized := make([]OptimizedMatch, len(dirs))
	for i, d := range dirs {
		optimized[i] = OptimizedMatch{
			Dir:      d.Directory,
			Severity: d.Severity,
			Action:   d.Action,
		}
	}
	return optimized
}

func optimizeFilePatterns(patterns []FilePatternType) []OptimizedMatch {
	optimized := make([]OptimizedMatch, len(patterns))
	for i, p := range patterns {
		optimized[i] = OptimizedMatch{
			Pattern:  p.Pattern,
			Severity: p.Severity,
			Action:   p.Action,
		}
	}
	return optimized
}

type HostSecuritySpec struct {
	NodeSelector NodeSelectorType `json:"nodeSelector" yaml:"nodeSelector"`

	Process      ProcessType      `json:"process,omitempty" yaml:"process,omitempty"`
	File         FileType         `json:"file,omitempty" yaml:"file,omitempty"`
	Network      NetworkType      `json:"network,omitempty" yaml:"network,omitempty"`
	Capabilities CapabilitiesType `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Syscalls     SyscallsType     `json:"syscalls,omitempty" yaml:"syscalls,omitempty"`

	AppArmor string `json:"apparmor,omitempty" yaml:"apparmor,omitempty"`

	Severity int      `json:"severity" yaml:"severity"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action" yaml:"action"`
}

type NodeSelectorType struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty" yaml:"matchLabels,omitempty"`
	Identities  []string          `json:"identities,omitempty" yaml:"identities,omitempty"`
}

type ProcessType struct {
	MatchPaths       []ProcessPathType      `json:"matchPaths,omitempty" yaml:"matchPaths,omitempty"`
	MatchDirectories []ProcessDirectoryType `json:"matchDirectories,omitempty" yaml:"matchDirectories,omitempty"`
	MatchPatterns    []ProcessPatternType   `json:"matchPatterns,omitempty" yaml:"matchPatterns,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type ProcessPathType struct {
	Path       string            `json:"path,omitempty" yaml:"path,omitempty"`
	ExecName   string            `json:"execname,omitempty" yaml:"execname,omitempty"`
	OwnerOnly  bool              `json:"ownerOnly,omitempty" yaml:"ownerOnly,omitempty"`
	FromSource []MatchSourceType `json:"fromSource,omitempty" yaml:"fromSource,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type MatchSourceType struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

type ProcessDirectoryType struct {
	Directory  string            `json:"dir" yaml:"dir"`
	Recursive  bool              `json:"recursive,omitempty" yaml:"recursive,omitempty"`
	OwnerOnly  bool              `json:"ownerOnly,omitempty" yaml:"ownerOnly,omitempty"`
	FromSource []MatchSourceType `json:"fromSource,omitempty" yaml:"fromSource,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type FileType struct {
	MatchPaths       []FilePathType      `json:"matchPaths,omitempty" yaml:"matchPaths,omitempty"`
	MatchDirectories []FileDirectoryType `json:"matchDirectories,omitempty" yaml:"matchDirectories,omitempty"`
	MatchPatterns    []FilePatternType   `json:"matchPatterns,omitempty" yaml:"matchPatterns,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type FilePathType struct {
	Path       string            `json:"path" yaml:"path"`
	ReadOnly   bool              `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	OwnerOnly  bool              `json:"ownerOnly,omitempty" yaml:"ownerOnly,omitempty"`
	FromSource []MatchSourceType `json:"fromSource,omitempty" yaml:"fromSource,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type FileDirectoryType struct {
	Directory  string            `json:"dir" yaml:"dir"`
	ReadOnly   bool              `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	Recursive  bool              `json:"recursive,omitempty" yaml:"recursive,omitempty"`
	OwnerOnly  bool              `json:"ownerOnly,omitempty" yaml:"ownerOnly,omitempty"`
	FromSource []MatchSourceType `json:"fromSource,omitempty" yaml:"fromSource,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type FilePatternType struct {
	Pattern   string `json:"pattern" yaml:"pattern"`
	ReadOnly  bool   `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	OwnerOnly bool   `json:"ownerOnly,omitempty" yaml:"ownerOnly,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type NetworkType struct {
	MatchProtocols []NetworkProtocolType `json:"matchProtocols,omitempty" yaml:"matchProtocols,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type NetworkProtocolType struct {
	Protocol   string            `json:"protocol" yaml:"protocol"`
	FromSource []MatchSourceType `json:"fromSource,omitempty" yaml:"fromSource,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type CapabilitiesCapabilityType struct {
	Capability string            `json:"capability" yaml:"capability"`
	FromSource []MatchSourceType `json:"fromSource,omitempty" yaml:"fromSource,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type CapabilitiesType struct {
	MatchCapabilities []CapabilitiesCapabilityType `json:"matchCapabilities,omitempty" yaml:"matchCapabilities,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}

type SyscallsType struct {
	MatchSyscalls []SyscallMatchType     `json:"matchSyscalls,omitempty" yaml:"matchSyscalls,omitempty"`
	MatchPaths    []SyscallMatchPathType `json:"matchPaths,omitempty" yaml:"matchPaths,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
}

type SyscallMatchType struct {
	Syscalls   []string                `json:"syscall,omitempty" yaml:"syscall,omitempty"`
	FromSource []SyscallFromSourceType `json:"fromSource,omitempty" yaml:"fromSource,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
}

type SyscallMatchPathType struct {
	Path       string                  `json:"path,omitempty" yaml:"path,omitempty"`
	Recursive  bool                    `json:"recursive,omitempty" yaml:"recursive,omitempty"`
	Syscalls   []string                `json:"syscall,omitempty" yaml:"syscall,omitempty"`
	FromSource []SyscallFromSourceType `json:"fromSource,omitempty" yaml:"fromSource,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
}

type SyscallFromSourceType struct {
	Path      string `json:"path,omitempty" yaml:"path,omitempty"`
	Dir       string `json:"dir,omitempty" yaml:"dir,omitempty"`
	Recursive bool   `json:"recursive,omitempty" yaml:"recursive,omitempty"`
}

type ProcessPatternType struct {
	Pattern   string `json:"pattern" yaml:"pattern"`
	OwnerOnly bool   `json:"ownerOnly,omitempty" yaml:"ownerOnly,omitempty"`

	Severity int      `json:"severity,omitempty" yaml:"severity,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Message  string   `json:"message,omitempty" yaml:"message,omitempty"`
	Action   string   `json:"action,omitempty" yaml:"action,omitempty"`
}
