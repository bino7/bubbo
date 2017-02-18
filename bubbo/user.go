package bubbo

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/asdine/storm"
	"time"
)

type User struct {
	Username string `storm:"id"`
	Title    string
	Desc     string
	Email    string
	Password string
	Avatar   string
	Version  int64
}

func user(username string) (u *User, err error) {
	u = new(User)
	err = db.One("Username", username, u)
	if err == storm.ErrNotFound {
		err = ErrorUserNotFound
	}
	return
}

func authUser(u *User) (user *User, err error) {
	user = new(User)
	if u.Email != "" {
		err = db.One("Email", u.Email, user)
	} else if u.Username != "" {
		err = db.One("Username", u.Username, user)
	}

	if err != nil && err == storm.ErrNotFound {
		err = ErrorUserNotFound
	}
	if err == nil && user.Password != u.Password {
		err = ErrorForbidden
	}

	if !(err == nil && user.Password == u.Password) {
		user = nil
	}

	return
}

func registerUser(user *User) (err error) {
	if user.Username == "" || user.Email == "" || user.Password == "" {
		return ErrorEmptyField
	}

	u, err := authUser(user)

	if u != nil || err == ErrorForbidden {
		return ErrorUserExisted
	}

	user.Version = time.Now().Unix()

	err = db.Save(user)

	return
}

func updateUser(user *User) (err error) {
	user.Version = time.Now().Unix()
	err = db.Update(user)
	return
}

type following struct {
	username string `storm:"id"`
}

func (u *User) following() (following []string, err error) {
	flowgs := make([]*following, 0)
	node := db.From(u.Username, "following")
	err = node.All(flowgs)
	if err == nil {
		following = make([]string, len(flowgs))
		l := 0
		for _, f := range flowgs {
			u, err = user(f.username)
			if u != nil && err == nil {
				following[l] = f.username
				l++
			}
		}
		following = following[:l - 1]
	}
	return
}

func (u *User) follow(username string) (err error) {
	user, err = user(username)
	if user == nil && err != nil {
		return
	}

	tx, err := db.Begin(true)
	u.Version = time.Now().Unix()
	err = db.UpdateField(u, "Version", u.Version)
	if err != nil {
		return
	}
	n := db.From(u.Username, "following")
	n = n.WithTransaction(tx)
	err = n.Save(&following{username})
	if err != nil {
		return
	}
	return tx.Commit()
}

func (u *User) unfollow(username string) (err error) {
	user, err = user(username)
	if user == nil && err != nil {
		return
	}

	tx, err := db.Begin(true)
	u.Version = time.Now().Unix()
	err = db.UpdateField(u, "Version", u.Version)
	if err != nil {
		return
	}
	n := db.From(u.Username, "following")
	n = n.WithTransaction(tx)
	err = n.Drop(&following{username})
	if err != nil {
		return
	}
	return tx.Commit()
}

type follower struct {
	username string `storm:"id"`
}

func (u *User) followers() (followers []string, err error) {
	flowgs := make([]*follower, 0)
	node := db.From(u.Username, "follower")
	err = node.All(flowgs)
	if err == nil {
		followers = make([]string, len(flowgs))
		l := 0
		for _, f := range flowgs {
			u, err = user(f.username)
			if u != nil && err == nil {
				followers[l] = f.username
				l++
			}
		}
		followers = followers[:l - 1]
	}
	return
}

func (u *User) addFollower(username string) (err error) {
	user, err = user(username)
	if user == nil && err != nil {
		return
	}

	tx, err := db.Begin(true)
	u.Version = time.Now().Unix()
	err = db.UpdateField(u, "Version", u.Version)
	if err != nil {
		return
	}
	n := db.From(u.Username, "follower")
	n = n.WithTransaction(tx)
	err = n.Save(&follower{username})
	if err != nil {
		return
	}
	return tx.Commit()
}

func (u *User) removeFollower(username string) (err error) {
	user, err = user(username)
	if user == nil && err != nil {
		return
	}

	tx, err := db.Begin(true)
	u.Version = time.Now().Unix()
	err = db.UpdateField(u, "Version", u.Version)
	if err != nil {
		return
	}
	n := db.From(u.Username, "follower")
	n = n.WithTransaction(tx)
	err = n.Drop(&follower{username})
	if err != nil {
		return
	}
	return tx.Commit()
}

/*type Media struct {
	name string `storm:"id"`
	data string
}

func (u *User) medias() (medias []Media, err error) {
	medias = make([]*Media, 0)
	node := db.From(u.Username, "media")
	err = node.All(medias)
	return
}

func (u *User) addMedia(media Media) (err error) {
	tx, err := db.Begin(true)
	u.Version = time.Now().Unix()
	err = db.UpdateField(u, "Version", u.Version)
	if err != nil {
		return
	}
	n := db.From(u.Username, "follower")
	n = n.WithTransaction(tx)
	err = n.Save(&follower{username})
	if err != nil {
		return
	}
	return tx.Commit()
}

func (u *User) removeMedia(username string) (err error) {
	user, err = user(username)
	if user == nil && err != nil {
		return
	}

	tx, err := db.Begin(true)
	u.Version = time.Now().Unix()
	err = db.UpdateField(u, "Version", u.Version)
	if err != nil {
		return
	}
	n := db.From(u.Username, "follower")
	n = n.WithTransaction(tx)
	err = n.Drop(&follower{username})
	if err != nil {
		return
	}
	return tx.Commit()
}*/

func (u *User) token(uuid string) *jwt.Token {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"UUID": uuid,
		"Username": u.Username,
		"Password":u.Password,
		"Email":u.Email,
	})
	return token
}

type UserProfile struct {
	Username string
	Title    string
	Desc     string
	Email    string
	Avatar   string
	Version  int64
}

func (u *User) profile() (*UserProfile) {
	return &UserProfile{
		Username:u.Username,
		Title:u.Title,
		Desc:u.Desc,
		Email:u.Email,
		Avatar:u.Avatar,
		Version:u.Version,
	}
}



