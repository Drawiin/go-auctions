package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection         *mongo.Collection
	auctionDuration    time.Duration
	auctionStatusMutex *sync.Mutex
}

func NewAuctionRepository(database *mongo.Database, auctionStatusMutex *sync.Mutex) *AuctionRepository {
	return &AuctionRepository{
		Collection:         database.Collection("auctions"),
		auctionDuration:    getAuctionDuration(),
		auctionStatusMutex: auctionStatusMutex,
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		// We add the duration to the current time to set the auction end time
		Timestamp: auctionEntity.Timestamp.Add(ar.auctionDuration).Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	go func() {
		time.Sleep(ar.auctionDuration)
		// We lock to make sure no readers are accessing the status while were updating it
		ar.auctionStatusMutex.Lock()
		_, err := ar.CloseAuction(ctx, auctionEntity.Id)
		ar.auctionStatusMutex.Unlock()
		if err != nil {
			logger.Error("Error trying to close auction", err)
		}
		fmt.Println("Auction closed", auctionEntity.Id)
	}()

	return nil
}

func getAuctionDuration() time.Duration {
	auctionInterval := os.Getenv("AUCTION_DURATION")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Minute * 6
	}

	return duration
}
