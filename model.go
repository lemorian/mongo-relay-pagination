package relayconnectionspec

type PageInfo struct {
	HasNextPage     bool    `json:"hasNextPage" bson:"hasNextPage"`
	HasPreviousPage bool    `json:"hasPreviousPage" bson:"hasPreviousPage"`
	StartCursor     *string `json:"startCursor" bson:"startCursor,omitempty"`
	EndCursor       *string `json:"endCursor" bson:"endCursor,omitempty"`
}

type Node interface {
	IsNode()
}

type Connection struct {
	Edges      []*Edge   `json:"edges" bson:"edges"`
	PageInfo   *PageInfo `json:"pageInfo" bson:"pageInfo,omitempty"`
	TotalCount int64     `json:"totalCount" bson:"totalCount"`
}

type Edge struct {
	Node   *interface{} `json:"node" bson:"node,omitempty"`
	Cursor *string      `json:"cursor" bson:"cursor,omitempty"`
}
