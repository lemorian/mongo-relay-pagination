package relay

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Error constants
const (
	PageLimitError         = "first and last cannot be less than 0"
	DecodeEmptyError       = "struct should be provide to decode data"
	DecodeNotAvail         = "this feature is not available for aggregate query"
	FilterInAggregateError = "you cannot use filter in aggregate query but you can pass multiple filter as param in aggregate function"
	NilFilterError         = "filter query cannot be nil"
)

type PageInfo struct {
	HasNextPage     bool    `json:"hasNextPage" bson:"hasNextPage"`
	HasPreviousPage bool    `json:"hasPreviousPage" bson:"hasPreviousPage"`
	StartCursor     *string `json:"startCursor" bson:"startCursor,omitempty"`
	EndCursor       *string `json:"endCursor" bson:"endCursor,omitempty"`
}

type RelayConnectionCreator interface {
	CreateEdge(cursorDecoder func(val interface{}) error) (string, error)
	SetTotalCount(int64)
	SetPageInfo(PageInfo)
}

//PagionationOptions - Arguments received from the GraphQL Client
type Options struct {
	First   int64
	Last    int64
	After   string
	Before  string
	Filter  *bson.M
	Project *bson.M
}
type Paginator struct {
	Collection    *mongo.Collection
	Ctx           context.Context
	ConCreator    RelayConnectionCreator
	PagingOptions Options
}

func mergeFilters(cursorFilter *bson.M, searchFilter *bson.M) bson.M {
	var mergedFilter = bson.M{}

	if cursorFilter != nil {
		for key, element := range *cursorFilter {
			mergedFilter[key] = element
		}
	}

	if searchFilter != nil {
		for key, element := range *searchFilter {
			mergedFilter[key] = element
		}
	}

	return mergedFilter
}

// validateQuery query is to check if user has added certain required params or not
func (paginator *Paginator) validateQuery(isNormal bool) error {
	if paginator.PagingOptions.First <= 0 && paginator.PagingOptions.Last <= 0 {
		return errors.New(PageLimitError)
	}
	if isNormal && paginator.ConCreator == nil {
		return errors.New(DecodeEmptyError)
	}
	if !isNormal && paginator.ConCreator != nil {
		return errors.New(DecodeNotAvail)
	}
	return nil
}

func (paginator *Paginator) getContext() context.Context {
	if paginator.Ctx != nil {
		return paginator.Ctx
	} else {
		return context.Background()
	}
}

func (paginator *Paginator) setTotal() error {

	ctx := paginator.getContext()
	var total int64
	var err error
	if paginator.PagingOptions.Filter != nil {
		total, err = paginator.Collection.CountDocuments(ctx, paginator.PagingOptions.Filter)
	} else {
		total, err = paginator.Collection.CountDocuments(ctx, bson.M{})
	}
	if err != nil {
		return err
	}
	paginator.ConCreator.SetTotalCount(total)
	return nil
}

func (paginator *Paginator) hasMore(cursorFilter bson.M, searchFilter *bson.M, limit int64) (bool, error) {
	ctx := paginator.getContext()
	var mergedFilter = mergeFilters(&cursorFilter, searchFilter)

	resultCount, err := paginator.Collection.CountDocuments(ctx, mergedFilter)

	if err != nil {
		return false, err
	}

	if resultCount >= limit {
		return true, nil
	}

	return false, nil
}

func (paginator *Paginator) Find() error {

	if err := paginator.validateQuery(true); err != nil {
		return err
	}

	opts := &options.FindOptions{}
	var hasNextPageLimit int64

	if paginator.PagingOptions.First > 0 {
		opts.SetLimit(paginator.PagingOptions.First)
		hasNextPageLimit = paginator.PagingOptions.First + 1
	} else {
		opts.SetLimit(paginator.PagingOptions.Last)
		hasNextPageLimit = paginator.PagingOptions.Last + 1
	}

	var cursorFilter bson.M = bson.M{}
	var cursorFilterReverse bson.M = bson.M{}

	//If Both after and before is set, only after is considered
	if len(paginator.PagingOptions.After) > 0 {
		oid, err := primitive.ObjectIDFromHex(paginator.PagingOptions.After)
		if err != nil {
			return err
		}
		cursorFilter["_id"] = bson.M{"$gt": oid}
		cursorFilterReverse["_id"] = bson.M{"$lt": oid}
	} else if len(paginator.PagingOptions.Before) > 0 {
		oid, err := primitive.ObjectIDFromHex(paginator.PagingOptions.Before)
		if err != nil {
			return err
		}
		cursorFilter["_id"] = bson.M{"$lt": oid}
		cursorFilterReverse["_id"] = bson.M{"$gt": oid}
	}

	ctx := paginator.getContext()

	err := paginator.setTotal()

	if err != nil {
		return err
	}

	var mergedFilter = mergeFilters(&cursorFilter, paginator.PagingOptions.Filter)

	mgoCursor, err := paginator.Collection.Find(ctx, mergedFilter, opts)

	if err != nil {
		return err
	}

	var pageInfo PageInfo = PageInfo{}
	var currentCursor string
	for mgoCursor.Next(ctx) {

		currentCursor, err := paginator.ConCreator.CreateEdge(mgoCursor.Decode)

		if err != nil {
			return err
		}

		if pageInfo.StartCursor == nil {
			pageInfo.StartCursor = &currentCursor
		}
		pageInfo.EndCursor = &currentCursor
	}

	pageInfo.HasNextPage, err = paginator.hasMore(cursorFilter, paginator.PagingOptions.Filter, hasNextPageLimit)

	if err != nil {
		return err
	}

	pageInfo.HasPreviousPage, err = paginator.hasMore(cursorFilterReverse, paginator.PagingOptions.Filter, 1)

	if err != nil {
		return err
	}

	paginator.ConCreator.SetPageInfo(pageInfo)

	return nil
}
