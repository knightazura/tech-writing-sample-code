package main

type Transaction struct {
	ID        int64    `json:"-"`
	UserID    int      `json:"user_id"`
	Name      string   `json:"name"`
	Items     []string `json:"items"`
	Amount    int64    `json:"amount"`
	CreatedAt int64    `json:"created_at,omitempty"`
}
