package user

// Storage interface: Mongo & Mock
type Storage interface {
	Close()
	Ping() error
	UpsertUser(user *User) error
	FindUserByID(user *User) (*User, error)
	FindUser(user *User) ([]*User, error)
	FindUsersByRole(role string) ([]*User, error)
	FindUsersWithIds(role []string) ([]*User, error)
	RemoveUser(user *User) error
}
