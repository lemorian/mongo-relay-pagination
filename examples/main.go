package main

import (
	"context"
	"log"
	"time"

	relay "github.com/lemorian/go-relay-connection"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Subject struct {
	ID       string `json:"id" bson:"_id"`
	Name     string `json:"name" bson:"name"`
	Overview string `json:"overview" bson:"overview"`
}
type SubjectEdge struct {
	Node   Subject `json:"node" bson:"node,omitempty"`
	Cursor string  `json:"cursor" bson:"cursor,omitempty"`
}

type SubjectConnection struct {
	Edges      []SubjectEdge  `json:"edges" bson:"edges"`
	PageInfo   relay.PageInfo `json:"pageInfo" bson:"pageInfo,omitempty"`
	TotalCount int64          `json:"totalCount" bson:"totalCount"`
}

func (connection *SubjectConnection) CreateEdge(decoder func(val interface{}) error) (string, error) {
	var subject *Subject = &Subject{}
	err := decoder(subject)
	if err != nil {
		return "", err
	}
	connection.Edges = append(connection.Edges, SubjectEdge{
		Node:   *subject,
		Cursor: subject.ID,
	})
	return subject.ID, nil
}

func (connection *SubjectConnection) SetTotalCount(count int64) {
	connection.TotalCount = count
}

func (connection *SubjectConnection) SetPageInfo(pageInfo relay.PageInfo) {
	connection.PageInfo = pageInfo
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	var db = client.Database("dev")

	var collection = db.Collection("subjects")

	var subjectCollection *SubjectConnection = &SubjectConnection{}

	paginator := relay.Paginator{
		Collection: collection,
		Ctx:        ctx,
		ConCreator: subjectCollection,
		PagingOptions: relay.Options{
			First: 5,
			After: "605db89d208db95eb4878553",
			Filter: &bson.M{
				"overview": "Tuesday",
			},
		},
	}

	err = paginator.Find()

	if err != nil {
		log.Println("error", err.Error())
	}

	// for _, result := range results {
	// 	if subject, ok := result.(SubjectEdge); ok {
	// 		subjects = append(subjects, subject)
	// 	}
	// }
	// if err != nil {
	// 	log.Println("error", err.Error())
	// }

	log.Printf("results %v", subjectCollection.PageInfo.HasNextPage)
}
