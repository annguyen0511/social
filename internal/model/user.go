package model

type User struct {
	ID        int64  `json:"id" db:"id"`
	FirstName string `json:"first_name" db:"first_name"`
	LastName  string `json:"last_name" db:"last_name"`
	AvatarURL string `json:"avatar_url" db:"avatar_url"`
	UserName  string `json:"username" db:"username"`
	Email     string `json:"email" db:"email"`
	Password  string `json:"-"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}
