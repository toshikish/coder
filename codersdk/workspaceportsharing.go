package codersdk

const (
	WorkspaceAgentPortSharingLevelOwner         WorkspacePortSharingLevel = 0
	WorkspaceAgentPortSharingLevelAuthenticated WorkspacePortSharingLevel = 1
	WorkspaceAgentPortSharingLevelPublic        WorkspacePortSharingLevel = 2
)

type (
	WorkspacePortSharingLevel                   int
	UpdateWorkspaceAgentPortSharingLevelRequest struct {
		AgentName  string `json:"agent_name"`
		Port       int32  `json:"port"`
		ShareLevel int32  `json:"share_level"`
	}
)