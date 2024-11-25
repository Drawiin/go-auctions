package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

func (ar *AuctionRepository) CloseAuction(
	ctx context.Context, id string) (*auction_entity.Auction, *internal_error.InternalError) {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"status": auction_entity.Completed}}

	var auctionEntityMongo AuctionEntityMongo
	if err := ar.Collection.FindOneAndUpdate(ctx, filter, update).Decode(&auctionEntityMongo); err != nil {
		logger.Error(fmt.Sprintf("Error trying to update auction status by id = %s", id), err)
		return nil, internal_error.NewInternalServerError("Error trying to update auction status by id")
	}

	return &auction_entity.Auction{
		Id:          auctionEntityMongo.Id,
		ProductName: auctionEntityMongo.ProductName,
		Category:    auctionEntityMongo.Category,
		Description: auctionEntityMongo.Description,
		Condition:   auctionEntityMongo.Condition,
		Status:      auction_entity.Completed,
		Timestamp:   time.Unix(auctionEntityMongo.Timestamp, 0),
	}, nil
}
