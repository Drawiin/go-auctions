package auction_test

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestCreateAuction(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("successfully create auction", func(mt *mtest.T) {
		auctionRepo := auction.NewAuctionRepository(mt.DB, &sync.Mutex{})
		auctionEntity := &auction_entity.Auction{
			Id:          "1",
			ProductName: "Test Product",
			Category:    "Test Category",
			Description: "Test Description",
			Condition:   auction_entity.New,
			Status:      auction_entity.Active,
			Timestamp:   time.Now(),
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse(), mtest.CreateCursorResponse(1, "test.auctions", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "1"},
			{Key: "product_name", Value: "Test Product"},
			{Key: "category", Value: "Test Category"},
			{Key: "description", Value: "Test Description"},
			{Key: "condition", Value: auction_entity.New},
			{Key: "status", Value: auction_entity.Active},
			{Key: "timestamp", Value: auctionEntity.Timestamp.Add(time.Minute * 6).Unix()},
		}))

		err := auctionRepo.CreateAuction(context.Background(), auctionEntity)
		assert.Nil(t, err)

		var result auction.AuctionEntityMongo
		err2 := mt.DB.Collection("auctions").FindOne(context.Background(), bson.M{"_id": "1"}).Decode(&result)
		assert.Nil(t, err2)
		assert.Equal(t, auctionEntity.Id, result.Id)
		assert.Equal(t, auctionEntity.ProductName, result.ProductName)
		assert.Equal(t, auctionEntity.Category, result.Category)
		assert.Equal(t, auctionEntity.Description, result.Description)
		assert.Equal(t, auctionEntity.Condition, result.Condition)
		assert.Equal(t, auctionEntity.Status, result.Status)
	})
}
