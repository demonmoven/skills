package domain

import "example.com/rotten/internal/infra"

func Load(id string) string {
	return infra.ReadDB(id)
}
