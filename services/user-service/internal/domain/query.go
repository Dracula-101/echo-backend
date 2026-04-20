package domain

type UserQueryType string

const (
	UserQueryTypeUsername UserQueryType = "username"
	UserQueryTypeName     UserQueryType = "name"
	UserQueryTypeAll      UserQueryType = "all"
)

func (t UserQueryType) IsValid() bool {
	switch t {
	case UserQueryTypeUsername, UserQueryTypeName, UserQueryTypeAll:
		return true
	default:
		return false
	}
}

func (t UserQueryType) String() string {
	return string(t)
}
