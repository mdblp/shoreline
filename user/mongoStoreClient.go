package user

import (
	"fmt"
	"log"
	"regexp"
	"sort"

	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	usersCollection = "users"
)

// Client struct
type Client struct {
	*goComMgo.StoreClient
}

// NewStore creates a new Client
func NewStore(config *goComMgo.Config, logger *log.Logger) (*Client, error) {
	client := Client{}
	store, err := goComMgo.NewStoreClient(config, logger)
	client.StoreClient = store
	return &client, err
}

func mgoUsersCollection(c *Client) *mongo.Collection {
	return c.Collection(usersCollection)
}

func (c *Client) UpsertUser(user *User) error {
	if user.Roles != nil {
		sort.Strings(user.Roles)
	}
	options := options.Update().SetUpsert(true)
	update := bson.D{{"$set", user}}
	// if the user already exists we update otherwise we add
	_, err := mgoUsersCollection(c).UpdateOne(c.Context, bson.M{"userid": user.ID}, update, options)
	return err
}

func (c *Client) FindUser(user *User) (result *User, err error) {
	if user.ID != "" {
		opts := options.FindOne()
		if err = mgoUsersCollection(c).FindOne(c.Context, bson.M{"userid": user.ID}, opts).Decode(&result); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (c *Client) findUsers(filter interface{}, noResultMessage string) (results []*User, err error) {
	cursor, err := mgoUsersCollection(c).Find(c.Context, filter)
	defer cursor.Close(c.Context)
	if err != nil {
		return results, err
	}
	err = cursor.All(c.Context, &results)
	if err != nil {
		return results, err
	}
	if results == nil {
		log.Print(noResultMessage)
		results = []*User{}
	}

	return results, nil
}

func (c *Client) FindUsers(user *User) (results []*User, err error) {
	fieldsToMatch := []bson.M{}

	if user.ID != "" {
		fieldsToMatch = append(fieldsToMatch, bson.M{"userid": user.ID})
	}
	if user.Username != "" {
		regexFilter := primitive.Regex{Pattern: fmt.Sprintf(`^%s$`, regexp.QuoteMeta(user.Username)), Options: "i"}
		fieldsToMatch = append(fieldsToMatch, bson.M{"username": bson.M{"$regex": regexFilter}})
	}
	if len(user.Emails) > 0 {
		fieldsToMatch = append(fieldsToMatch, bson.M{"emails": bson.M{"$in": user.Emails}})
	}

	if len(fieldsToMatch) == 0 {
		return []*User{}, nil
	}
	noUserMessage := fmt.Sprintf("no users found: query: (Id = %v) OR (Name ~= %v) OR (Emails IN %v)", user.ID, user.Username, user.Emails)
	return c.findUsers(bson.M{"$or": fieldsToMatch}, noUserMessage)
}

func (c *Client) FindUsersByRole(role string) (results []*User, err error) {
	noUserMessage := fmt.Sprintf("no users found: query: role: %v", role)
	return c.findUsers(bson.M{"roles": role}, noUserMessage)
}

func (c *Client) FindUsersWithIds(ids []string) (results []*User, err error) {
	noUserMessage := fmt.Sprintf("no users found: query: id: %v", ids)
	return c.findUsers(bson.M{"userid": bson.M{"$in": ids}}, noUserMessage)
}

func (c *Client) RemoveUser(user *User) (err error) {
	if _, err := mgoUsersCollection(c).DeleteOne(c.Context, bson.M{"userid": user.ID}); err != nil {
		return err
	}
	return nil
}
