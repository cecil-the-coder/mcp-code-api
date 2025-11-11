package api

type ClientMetadata struct {
	IDEType       string `json:"ideType,omitempty"`
	IDEVersion    string `json:"ideVersion,omitempty"`
	PluginVersion string `json:"pluginVersion,omitempty"`
	Platform      string `json:"platform,omitempty"`
	UpdateChannel string `json:"updateChannel,omitempty"`
	DuetProject   string `json:"duetProject,omitempty"`
	PluginType    string `json:"pluginType,omitempty"`
	IDEName       string `json:"ideName,omitempty"`
}

const (
	IDETypeUnspecified  = "IDE_UNSPECIFIED"
	PlatformUnspecified = "PLATFORM_UNSPECIFIED"
	PluginTypeGemini    = "GEMINI"
)

type LoadCodeAssistRequest struct {
	CloudaicompanionProject *string        `json:"cloudaicompanionProject,omitempty"`
	Metadata                ClientMetadata `json:"metadata"`
}

type GeminiUserTier struct {
	ID                                 string         `json:"id"`
	Name                               string         `json:"name"`
	Description                        string         `json:"description"`
	UserDefinedCloudaicompanionProject *bool          `json:"userDefinedCloudaicompanionProject,omitempty"`
	IsDefault                          *bool          `json:"isDefault,omitempty"`
	PrivacyNotice                      *PrivacyNotice `json:"privacyNotice,omitempty"`
	HasAcceptedTos                     *bool          `json:"hasAcceptedTos,omitempty"`
	HasOnboardedPreviously             *bool          `json:"hasOnboardedPreviously,omitempty"`
}

type LoadCodeAssistResponse struct {
	CurrentTier             *GeminiUserTier  `json:"currentTier,omitempty"`
	AllowedTiers            []GeminiUserTier `json:"allowedTiers,omitempty"`
	IneligibleTiers         []IneligibleTier `json:"ineligibleTiers,omitempty"`
	CloudaicompanionProject *string          `json:"cloudaicompanionProject,omitempty"`
}

type PrivacyNotice struct {
	ShowNotice bool    `json:"showNotice"`
	NoticeText *string `json:"noticeText,omitempty"`
}

type IneligibleTier struct {
	ReasonCode    string `json:"reasonCode"`
	ReasonMessage string `json:"reasonMessage"`
	TierID        string `json:"tierId"`
	TierName      string `json:"tierName"`
}

type OnboardUserRequest struct {
	TierID                  *string         `json:"tierId,omitempty"`
	CloudaicompanionProject *string         `json:"cloudaicompanionProject,omitempty"`
	Metadata                *ClientMetadata `json:"metadata,omitempty"`
}

type OnboardUserResponse struct {
	CloudaicompanionProject *CloudaicompanionProject `json:"cloudaicompanionProject,omitempty"`
}

type LongRunningOperationResponse struct {
	Name     string               `json:"name"`
	Done     bool                 `json:"done"`
	Response *OnboardUserResponse `json:"response,omitempty"`
}

type CloudaicompanionProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const (
	UserTierIDFree     = "free-tier"
	UserTierIDLegacy   = "legacy-tier"
	UserTierIDStandard = "standard-tier"
)
