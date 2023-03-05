package models

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-password/password"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string
	Password []byte
	Token    string
}

func userFromBytes(data []byte) (*User, error) {
	var p *User
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func AuthenticateLocalUser(incoming User) (bool, error) {
	current, err := GetUser(incoming.Username)
	if err != nil {
		return false, err
	}
	if err := bcrypt.CompareHashAndPassword(current.Password, incoming.Password); err != nil {
		return false, err
	}
	return true, nil
}

func GetUser(username string) (User, error) {
	var existing []byte
	db.BoltDB.View(func(tx *bolt.Tx) error {
		userBytes := tx.Bucket([]byte(db.USER_BUCKET)).Get([]byte(username))
		existing = userBytes
		return nil
	})
	if len(existing) == 0 {
		return User{}, nil
	}
	user, err := userFromBytes(existing)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal user from bytes")
		return User{}, err
	}
	return *user, nil
}

func GetUsers() ([]User, error) {
	var users []User
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.USER_BUCKET))
		b.ForEach(func(k, v []byte) error {
			user, err := userFromBytes(v)
			if err != nil {
				return err
			}
			users = append(users, *user)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func CreateAdminUserIfNotExists() error {
	user, err := GetUser("admin")
	if err != nil {
		panic(err)
	}
	if user.Username != "" && len(user.Password) > 0 {
		if config.Env.AdminPassword == "" {
			return nil
		}
		pw := config.Env.AdminPassword
		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(pw)); err != nil {
			log.Info().Msg("updating admin user password to $GEMFAST_ADMIN_PASSWORD")
		} else {
			return nil
		}
	}
	user = User{
		Username: "admin",
		Password: getAdminPassword(),
	}
	userBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("could not marshal user to json: %v", err)
	}
	err = db.BoltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(db.USER_BUCKET)).Put([]byte(user.Username), userBytes)
		if err != nil {
			return fmt.Errorf("could not set: %v", err)
		}
		return nil
	})
	return nil
}

func CreateLocalUsers() error {
	if config.Env.AddLocalUsers == "" {
		log.Trace().Msg("no local users to add")
		return nil
	}
	var usernames []string
	db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.USER_BUCKET))
		if b == nil {
			return fmt.Errorf("get bucket: FAILED")
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if string(k) != "admin" {
				usernames = append(usernames, string(k))
			}
		}
		return nil
	})

	m := make(map[string]bool)
	usersFromEnv := config.Env.AddLocalUsers
	pairs := strings.Split(usersFromEnv, ",")
	db.BoltDB.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.USER_BUCKET))
		for _, pair := range pairs {
			u := strings.Split(pair, ":")
			username := u[0]
			pw := u[1]
			pwbytes, err := bcrypt.GenerateFromPassword([]byte(pw), 14)
			if err != nil {
				panic(err)
			}
			userToAdd := User{
				Username: username,
				Password: pwbytes,
			}
			m[username] = true
			userBytes, err := json.Marshal(userToAdd)
			if err != nil {
				return fmt.Errorf("could not marshal user to json: %v", err)
			}
			log.Trace().Str("username", username).Msg("added or modified user")
			b.Put([]byte(username), userBytes)
		}
		b = tx.Bucket([]byte(db.USER_BUCKET))
		for _, username := range usernames {
			if m[username] != true {
				log.Trace().Str("username", username).Msg("removed user")
				b.Delete([]byte(username))
			}
		}
		return nil
	})
	return nil
}

func getAdminPassword() []byte {
	var pw string
	var err error
	if config.Env.AdminPassword == "" {
		pw, err = generatePassword()
		if err != nil {
			panic(err)
		}
	} else {
		pw = config.Env.AdminPassword
	}
	pwbytes, err := bcrypt.GenerateFromPassword([]byte(pw), 14)
	if err != nil {
		panic(err)
	}
	return pwbytes
}

func generatePassword() (string, error) {
	pw, err := password.Generate(32, 10, 0, false, false)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate an admin password")
		return "", err
	}
	log.Warn().Msg("generating admin password because environment variable GEMFAST_ADMIN_PASSWORD not set")
	log.Info().Str("password", pw).Msg("generated admin password")
	return pw, nil
}

func CreateUserToken(user *User) (string, error) {
	token, err := password.Generate(32, 10, 10, false, false)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate a token")
		return "", err
	}
	user.Token = token
	userBytes, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("could not marshal user to json: %v", err)
	}
	err = db.BoltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(db.USER_BUCKET)).Put([]byte(user.Username), userBytes)
		if err != nil {
			return fmt.Errorf("could not set: %v", err)
		}
		return nil
	})
	return b64.StdEncoding.EncodeToString([]byte(token)), err
}
