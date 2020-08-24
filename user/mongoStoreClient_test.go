package user

import (
	"strings"
	"testing"

	"github.com/globalsign/mgo"

	"github.com/tidepool-org/go-common/clients/mongo"
)

func mgoTestSetup() (*MongoStoreClient, error) {
	mc := NewMongoStoreClient(&mongo.Config{ConnectionString: "mongodb://127.0.0.1/user_test"})

	/*
	 * INIT THE TEST - we use a clean copy of the collection before we start
	 */
	cpy := mc.session.Copy()
	defer cpy.Close()

	//just drop and don't worry about any errors
	mgoUsersCollection(cpy).DropCollection()

	if err := mgoUsersCollection(cpy).Create(&mgo.CollectionInfo{}); err != nil {
		return nil, err
	}
	return mc, nil
}

func TestMongoStoreUserOperations(t *testing.T) {

	var (
		usernameOriginal   = "test@foo.bar"
		usernameOther      = "other@foo.bar"
		password           = "myT35ter"
		originalUserDetail = &NewUserDetails{Username: &usernameOriginal, Emails: []string{usernameOriginal}, Password: &password}
		otherUserDetail    = &NewUserDetails{Username: &usernameOther, Emails: originalUserDetail.Emails, Password: &password}
	)

	const testsFakeSalt = "some fake salt for the tests"

	mc, err := mgoTestSetup()
	if err != nil {
		t.Fatalf("we initialise the test store %s", err.Error())
	}

	/*
	 * THE TESTS
	 */
	user, err := NewUser(originalUserDetail, testsFakeSalt)
	if err != nil {
		t.Fatalf("we could not create the user %v", err)
	}

	if err := mc.UpsertUser(user); err != nil {
		t.Fatalf("we could not upsert the user %v", err)
	}

	/*
	 * Find by Username
	 */

	toFindByOriginalName := &User{Username: user.Username}

	if found, err := mc.FindUser(toFindByOriginalName); err != nil {
		t.Fatalf("we could not find the the user by name: err[%v]", err)
	} else {
		if len(found) > 0 && found[0].Username != toFindByOriginalName.Username && found[0].Username != *originalUserDetail.Username {
			t.Fatalf("the user we found doesn't match what we asked for %v", found)
		}
	}
	//UPPER CASE
	byUpperName := &User{Username: strings.ToUpper(user.Username)}

	if found, err := mc.FindUser(byUpperName); err != nil {
		t.Fatalf("we could not find the the user by name: err[%v]", err)
	} else {
		if len(found) == 0 {
			t.Fatal("No users were found for ", byUpperName.Username)
		} else if strings.ToUpper(found[0].Username) != byUpperName.Username {
			t.Fatalf("the user we found doesn't match what we asked for %v", found)
		}
	}
	//lower case
	byLowerName := &User{Username: strings.ToLower(user.Username)}

	if found, err := mc.FindUser(byLowerName); err != nil {
		t.Fatalf("we could not find the the user by name: err[%v]", err)
	} else {
		if len(found) == 0 {
			t.Fatal("No users were found for ", byLowerName.Username)
		} else if strings.ToLower(found[0].Username) != byLowerName.Username {
			t.Fatalf("the user we found doesn't match what we asked for %v", found)
		}
	}

	//Do an update
	user.Username = "test user updated"

	if err := mc.UpsertUser(user); err != nil {
		t.Fatalf("we could not update the user %v", err)
	}

	//By Username
	toFindByName := &User{Username: user.Username}

	if found, err := mc.FindUser(toFindByName); err != nil {
		t.Fatalf("we could not find the the user by name: err[%v]", err)
	} else {
		if len(found) != 1 {
			t.Logf("results: %v ", found)
			t.Fatalf("there should only be 1 match be we found %v", len(found))
		}
		if found[0].Username != toFindByName.Username {
			t.Fatalf("the user we found doesn't match what we asked for %v", found)
		}
	}

	/*
	 * Find by Email
	 */

	//By Email
	byEmails := &User{Emails: user.Emails}

	if found, err := mc.FindUser(byEmails); err != nil {
		t.Fatalf("we could not find the the user by emails %v", byEmails)
	} else {
		if len(found) != 1 {
			t.Logf("results: %v ", found)
			t.Fatalf("there should only be 1 match be we found %v", len(found))
		}
		if found[0].Emails[0] != byEmails.Emails[0] {
			t.Fatalf("the user we found doesn't match what we asked for %v", found)
		}
	}

	//By Id
	toFindByID := &User{ID: user.ID}

	if found, err := mc.FindUserByID(toFindByID); err != nil {
		t.Fatalf("we could not find the the user by id err[%v]", err)
	} else {
		if found.ID != toFindByID.ID {
			t.Fatalf("the user we found doesn't match what we asked for %v", found)
		}
	}

	//Find many By Email - user and userTwo have the same emails addresses
	userTwo, err := NewUser(otherUserDetail, testsFakeSalt)
	if err != nil {
		t.Fatalf("we could not create the user %v", err)
	}

	if err := mc.UpsertUser(userTwo); err != nil {
		t.Fatalf("we could not upsert the user %v", err)
	}

	toMultipleByEmails := &User{Emails: user.Emails}

	if found, err := mc.FindUser(toMultipleByEmails); err != nil {
		t.Fatalf("we could not find the the users by emails %v", toMultipleByEmails)
	} else if len(found) != 2 {
		t.Logf("results: %v ", found)
		t.Fatalf("there should be 2 match's be we found %v", len(found))
	}

}

func TestMongoStore_FindUsersByRole(t *testing.T) {

	var (
		testsFakeSalt = "some fake salt for the tests"
		userOneName   = "test@foo.bar"
		userTwoName   = "test_two@foo.bar"
		userPw        = "my0th3rT35t"
		userOneDetail = &NewUserDetails{Username: &userOneName, Emails: []string{userOneName}, Password: &userPw}
		userTwoDetail = &NewUserDetails{Username: &userTwoName, Emails: []string{userTwoName}, Password: &userPw}
	)

	mc, err := mgoTestSetup()
	if err != nil {
		t.Fatalf("we initialise the test store %s", err.Error())
	}

	/*
	 * THE TESTS
	 */
	userOne, _ := NewUser(userOneDetail, testsFakeSalt)
	userOne.Roles = append(userOne.Roles, "clinic")

	userTwo, _ := NewUser(userTwoDetail, testsFakeSalt)

	if err := mc.UpsertUser(userOne); err != nil {
		t.Fatalf("we could not create the user %v", err)
	}
	if err := mc.UpsertUser(userTwo); err != nil {
		t.Fatalf("we could not create the user %v", err)
	}

	if found, err := mc.FindUsersByRole("clinic"); err != nil {
		t.Fatalf("error finding users by role %s", err.Error())
	} else if len(found) != 1 || found[0].Roles[0] != "clinic" {
		t.Fatalf("should only find clinic users but found %v", found)
	}

}

func TestMongoStore_FindUsersById(t *testing.T) {

	var (
		testsFakeSalt = "some fake salt for the tests"
		userOneName   = "test@foo.bar"
		userTwoName   = "test_two@foo.bar"
		userPw        = "my0th3rT35t"
		userOneDetail = &NewUserDetails{Username: &userOneName, Emails: []string{userOneName}, Password: &userPw}
		userTwoDetail = &NewUserDetails{Username: &userTwoName, Emails: []string{userTwoName}, Password: &userPw}
	)

	mc, err := mgoTestSetup()
	if err != nil {
		t.Fatalf("we could not initialise the test store %s", err.Error())
	}

	/*
	 * THE TESTS
	 */
	userOne, _ := NewUser(userOneDetail, testsFakeSalt)
	userTwo, _ := NewUser(userTwoDetail, testsFakeSalt)

	if err := mc.UpsertUser(userOne); err != nil {
		t.Fatalf("we could not create the user %v", err)
	}
	if err := mc.UpsertUser(userTwo); err != nil {
		t.Fatalf("we could not create the user %v", err)
	}

	if found, err := mc.FindUsersWithIds([]string{userOne.ID}); err != nil {
		t.Fatalf("error finding users by role %s", err.Error())
	} else if len(found) != 1 || found[0].ID != userOne.ID {
		t.Fatalf("should only find user ID %s but found %v", userOne.ID, found)
	}

	if found, err := mc.FindUsersWithIds([]string{userTwo.ID}); err != nil {
		t.Fatalf("error finding users by role %s", err.Error())
	} else if len(found) != 1 || found[0].ID != userTwo.ID {
		t.Fatalf("should only find user ID %s but found %v", userTwo.ID, found)
	}

	if found, err := mc.FindUsersWithIds([]string{userOne.ID, userTwo.ID}); err != nil {
		t.Fatalf("error finding users by role %s", err.Error())
	} else if len(found) != 2 || found[0].ID != userOne.ID || found[1].ID != userTwo.ID {
		t.Fatalf("should only find user ID %s but found %v", userTwo.ID, found)
	}
}
