package bid

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/entity/bid_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
	"fullcycle-auction_go/internal/internal_error"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type BidEntityMongo struct {
	Id        string  `bson:"_id"`
	UserId    string  `bson:"user_id"`
	AuctionId string  `bson:"auction_id"`
	Amount    float64 `bson:"amount"`
	Timestamp int64   `bson:"timestamp"`
}

type BidRepository struct {
	Collection            *mongo.Collection
	AuctionRepository     *auction.AuctionRepository
	auctionStatusMap      map[string]auction_entity.AuctionStatus
	auctionEndTimeMap     map[string]time.Time
	auctionStatusMapMutex *sync.Mutex
	auctionEndTimeMutex   *sync.Mutex
	auctionStatusMutex    *sync.Mutex
}

func NewBidRepository(database *mongo.Database, auctionRepository *auction.AuctionRepository, mutex *sync.Mutex) *BidRepository {
	return &BidRepository{
		auctionStatusMap:      make(map[string]auction_entity.AuctionStatus),
		auctionEndTimeMap:     make(map[string]time.Time),
		auctionStatusMapMutex: &sync.Mutex{},
		auctionEndTimeMutex:   &sync.Mutex{},
		Collection:            database.Collection("bids"),
		AuctionRepository:     auctionRepository,
		auctionStatusMutex:    mutex,
	}
}

func (bd *BidRepository) CreateBid(
	ctx context.Context,
	bidEntities []bid_entity.Bid) *internal_error.InternalError {
	fmt.Println("Insert bid batch", len(bidEntities), bidEntities)
	var wg sync.WaitGroup
	for _, bid := range bidEntities {
		wg.Add(1)
		go func(bidValue bid_entity.Bid) {
			defer wg.Done()

			bd.auctionStatusMapMutex.Lock()
			auctionStatus, okStatus := bd.auctionStatusMap[bidValue.AuctionId]
			bd.auctionStatusMapMutex.Unlock()

			bd.auctionEndTimeMutex.Lock()
			auctionEndTime, okEndTime := bd.auctionEndTimeMap[bidValue.AuctionId]
			bd.auctionEndTimeMutex.Unlock()

			bidEntityMongo := &BidEntityMongo{
				Id:        bidValue.Id,
				UserId:    bidValue.UserId,
				AuctionId: bidValue.AuctionId,
				Amount:    bidValue.Amount,
				Timestamp: bidValue.Timestamp.Unix(),
			}

			if okEndTime && okStatus {
				now := time.Now()
				fmt.Println("bind auctionStatus", auctionStatus, "now", now, "auctionEndTime", auctionEndTime)
				if auctionStatus == auction_entity.Completed || now.After(auctionEndTime) {
					fmt.Println("bind droped auction completed or time expired", bidEntityMongo)
					return
				}

				if _, err := bd.Collection.InsertOne(ctx, bidEntityMongo); err != nil {
					logger.Error("Error trying to insert bid", err)
					return
				}

				fmt.Println("bind inserted", bidEntityMongo)
				return
			}

			// We lock the auction status to make sure we are not reading the status while it is being updated
			bd.auctionStatusMutex.Lock()
			auctionEntity, err := bd.AuctionRepository.FindAuctionById(ctx, bidValue.AuctionId)
			bd.auctionStatusMutex.Unlock()
			if err != nil {
				logger.Error("Error trying to find auction by id", err)
				return
			}
			if auctionEntity.Status == auction_entity.Completed {
				fmt.Println("bind droped auction completed ", bidEntityMongo)
				return
			}

			bd.auctionStatusMapMutex.Lock()
			bd.auctionStatusMap[bidValue.AuctionId] = auctionEntity.Status
			bd.auctionStatusMapMutex.Unlock()

			bd.auctionEndTimeMutex.Lock()
			// TODO - Remove this add since now the auction end time is set in the CreateAuction method
			bd.auctionEndTimeMap[bidValue.AuctionId] = auctionEntity.Timestamp
			bd.auctionEndTimeMutex.Unlock()

			if _, err := bd.Collection.InsertOne(ctx, bidEntityMongo); err != nil {
				logger.Error("Error trying to insert bid", err)
				return
			}
			fmt.Println("bind inserted", bidEntityMongo)
		}(bid)
	}
	wg.Wait()
	return nil
}
