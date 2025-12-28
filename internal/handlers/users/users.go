package users

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/k-tatiana/otus-project/models"
	"github.com/k-tatiana/otus-project/transport/postgres"
	"go.uber.org/zap"
)

type UserActions interface {
	GetUser(w http.ResponseWriter, r *http.Request)
	GetUsers(w http.ResponseWriter, r *http.Request)
	CreateUser(w http.ResponseWriter, r *http.Request)
	DeleteUser(w http.ResponseWriter, r *http.Request)
}

type UsersHandler struct {
	db     *postgres.DB
	logger *zap.Logger
}

func NewUsersHandler(db *postgres.DB, logger *zap.Logger) *UsersHandler {
	return &UsersHandler{db: db, logger: logger}
}

func (u *UsersHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User

	rows, err := u.db.Slave.Query(context.Background(), `SELECT id, first_name, last_name FROM customers`)
	if err != nil {
		u.logger.Error("failed to query customers", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName)
		if err != nil {
			u.logger.Error("failed to scan user row", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		u.logger.Error("error iterating rows", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(users)
	if err != nil {
		u.logger.Error("failed to encode users", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (u *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		u.logger.Error("failed to decode user", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	row, err := u.db.Master.Query(context.Background(), `INSERT INTO customers (first_name, last_name) VALUES ($1, $2)`, user.FirstName, user.LastName)
	if err != nil {
		u.logger.Error("failed to create user", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer row.Close()

	w.WriteHeader(http.StatusCreated)
}

func (u *UsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	var user models.User

	row := u.db.Slave.QueryRow(context.Background(), `SELECT first_name, last_name FROM customers WHERE id = $1`, r.URL.Query().Get("id"))
	if err := row.Scan(&user.FirstName, &user.LastName); err != nil {
		u.logger.Error("failed to scan user row", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(user)
	if err != nil {
		u.logger.Error("failed to encode user", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusFound)
}

func (u *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	_, err := u.db.Master.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, r.URL.Query().Get("id"))
	if err != nil {
		u.logger.Error("failed to delete user", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
