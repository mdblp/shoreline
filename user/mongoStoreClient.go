package user

import (
	"context"
	"fmt"
	"log"
	"time"

	mongoCommon "github.com/mdblp/go-common/clients/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	usersDatabase   = "user"
	usersCollection = "users"
	contextTimeout  = 5 * time.Second
	continuousPing  = true
)

// MongoStoreClient holds the mongo information
type MongoStoreClient struct {
	client     *mongoCommon.Store
	collection *mongo.Collection
	logger     *log.Logger
}

// NewMongoStoreClient create a new mongo client
func NewMongoStoreClient(config *mongoCommon.Config, logger *log.Logger) (MongoStoreClient, error) {
	client, err := mongoCommon.Connect(config, logger)
	if err != nil {
		return MongoStoreClient{}, err
	}

	if continuousPing {
		go client.ContinuousPing(contextTimeout)
	}

	return MongoStoreClient{
		client: client,
		logger: logger,
	}, nil
}

func (s *MongoStoreClient) ensureCollection() {
	if s.collection == nil {
		s.collection = (*s.client).GetCollection(usersDatabase, usersCollection)
	}
}

// Ping the database
func (s *MongoStoreClient) Ping() error {
	return s.client.Ping()
}

// Close the mongo connexion
func (s *MongoStoreClient) Close() error {
	return s.client.Disconnect()
}

// UpsertUser update an existing user
func (s *MongoStoreClient) UpsertUser(user *User) error {
	var err error

	if continuousPing && !s.client.PingOK {
		return fmt.Errorf("db connection error")
	}

	s.ensureCollection()

	s.logger.Printf("Mongo: UpsertUser: %v", user)
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	singleResult := s.collection.FindOne(ctx, bson.M{"userid": user.Id})
	err = singleResult.Err()
	if err != nil && err == mongo.ErrNoDocuments {
		_, err = s.collection.InsertOne(ctx, user)

	} else {
		updateResult, err := s.collection.UpdateOne(ctx, bson.M{"userid": user.Id}, user)
		if err == nil && updateResult.MatchedCount != 1 {
			err = fmt.Errorf("No document found")
		}
	}

	return err
}

// FindUser by ID in the database
func (s *MongoStoreClient) FindUser(user *User) (*User, error) {
	if continuousPing && !s.client.PingOK {
		return nil, fmt.Errorf("db connection error")
	}

	s.ensureCollection()
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	s.logger.Printf("FindUser %s", user.Id)
	result := s.collection.FindOne(ctx, bson.M{"userid": user.Id})
	err := result.Err()
	if err != nil && err == mongo.ErrNoDocuments {
		s.logger.Printf("FindUser %s: %v", user.Id, err)
		return nil, nil
	} else if err != nil {
		s.logger.Printf("FindUser %s: %v", user.Id, err)
		return nil, err
	}
	userFound := &User{}
	result.Decode(userFound)
	s.logger.Printf("FindUser %s: %v", user.Id, userFound)
	return userFound, nil
}

// FindUsers in the database
func (s *MongoStoreClient) FindUsers(user *User) ([]*User, error) {
	if continuousPing && !s.client.PingOK {
		return nil, fmt.Errorf("db connection error")
	}

	fieldsToMatch := bson.A{}

	if user.Id != "" {
		fieldsToMatch = append(fieldsToMatch, bson.M{"userid": user.Id})
	}
	if user.Username != "" {
		// case insensitive match
		regex := primitive.Regex{
			Pattern: fmt.Sprintf("^%s$", user.Username),
			Options: "i",
		}
		fieldsToMatch = append(fieldsToMatch, bson.M{"username": bson.M{"$regex": regex}})
	}
	if len(user.Emails) > 0 {
		fieldsToMatch = append(fieldsToMatch, bson.M{"emails": bson.M{"$in": user.Emails}})
	}

	s.ensureCollection()

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{"$or": fieldsToMatch})
	return s.findUsersResults(ctx, cursor, err)
}

// FindUsersByRole search users by role
func (s *MongoStoreClient) FindUsersByRole(role string) ([]*User, error) {
	if continuousPing && !s.client.PingOK {
		return nil, fmt.Errorf("db connection error")
	}

	s.ensureCollection()
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{"roles": role})
	return s.findUsersResults(ctx, cursor, err)
}

// FindUsersWithIds search users with a list of IDs
func (s *MongoStoreClient) FindUsersWithIds(ids []string) ([]*User, error) {
	if continuousPing && !s.client.PingOK {
		return nil, fmt.Errorf("db connection error")
	}

	s.ensureCollection()
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{"ids": ids})
	return s.findUsersResults(ctx, cursor, err)
}

func (s *MongoStoreClient) findUsersResults(ctx context.Context, cursor *mongo.Cursor, err error) ([]*User, error) {
	if err != nil {
		s.logger.Printf("findUsersResults error %v", err)
		return nil, err
	}

	var users []User
	err = cursor.All(ctx, &users)

	if err != nil {
		return nil, err
	}

	s.logger.Printf("findUsersResults %d => %v", len(users), users)

	usersPtr := make([]*User, len(users))

	for index, user := range users {
		usersPtr[index] = &user
	}

	return usersPtr, nil
}

// RemoveUser from the database
func (s *MongoStoreClient) RemoveUser(user *User) error {
	if continuousPing && !s.client.PingOK {
		return fmt.Errorf("db connection error")
	}

	s.logger.Printf("Mongo: RemoveUser: %v", user)

	s.ensureCollection()

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	delResult, err := s.collection.DeleteOne(ctx, bson.M{"userid": user.Id})
	if err != nil {
		return err
	}

	if delResult.DeletedCount != 1 {
		return fmt.Errorf("Nothing deleted")
	}

	return nil
}
