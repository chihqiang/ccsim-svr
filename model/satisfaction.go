package model

import "time"

// SatisfactionRating 满意度评价
type SatisfactionRating struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	SessionID int64     `json:"sessionId" gorm:"uniqueIndex;not null;comment:会话ID"`
	VisitorID int64     `json:"visitorId" gorm:"not null;comment:访客ID"`
	AgentID   int64     `json:"agentId" gorm:"index;comment:客服ID"`
	Rating    int8      `json:"rating" gorm:"type:tinyint;not null;comment:评分 1:满意 2:一般 3:不满意"`
	Comment   string    `json:"comment" gorm:"type:varchar(500);default:'';comment:评价内容"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime;comment:创建时间"`
}

// TableName 表名
func (SatisfactionRating) TableName() string {
	return "ccsim_satisfaction_ratings"
}
