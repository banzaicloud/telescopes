package engine

type HollowGroupRequest struct {
	AutoScalingGroupName string `json:"autoScalingGroupName" binding:"required"`
}

type CloudEngine interface {
	CreateHollowGroup(group *HollowGroupRequest) (string, error)
}
