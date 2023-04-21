package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gemfast/server/internal/db"
	bolt "go.etcd.io/bbolt"
)

type Gem struct {
	Name     string `json:"name"`
	Number   string `json:"number"`
	Platform string `json:"platform"`
}

func GemFromGemParameter(param string) *Gem {
	var gemName []string
	var gemVersion string
	chunks := strings.Split(param, "-")
	l := len(chunks)
	for i, chunk := range chunks {
		if (i + 1) == l {
			gemVersion = chunk
			break
		}
		gemName = append(gemName, chunk)
	}
	return &Gem{
		Name:   strings.Join(gemName, "-"),
		Number: gemVersion,
	}
}

func GemFromBytes(data []byte) (*[]Gem, error) {
	var p *[]Gem
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func GetGem(name string) ([]Gem, error) {
	var gems []Gem
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		g := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Get([]byte(name))
		gem, _ := GemFromBytes(g)
		gems = *gem
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gems, nil

}

func GetGems() ([][]Gem, error) {
	var gems [][]Gem
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			g, _ := GemFromBytes(v)
			gems = append(gems, *g)
		}
		if gems == nil {
			return fmt.Errorf("no gems found")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gems, nil
}
